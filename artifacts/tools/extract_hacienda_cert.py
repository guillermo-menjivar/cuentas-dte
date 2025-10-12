#!/usr/bin/env python3
"""
Script to extract certificate from XML format and prepare it for Docker Firmador service.
Usage: python3 prepare_cert_for_firmador.py <xml_cert_file>
"""

import base64
import xml.etree.ElementTree as ET
import sys
import os
from pathlib import Path


def extract_cert_from_xml(xml_file):
    """Extract certificate components from XML and create files for firmador"""

    print(f"📄 Reading XML certificate: {xml_file}")

    # Read XML file
    with open(xml_file, "r", encoding="utf-8") as f:
        xml_content = f.read()

    # Parse XML
    root = ET.fromstring(xml_content)

    # Extract NIT
    nit = root.find(".//nit").text.strip()
    print(f"✓ Found NIT: {nit}")

    # Extract Base64-encoded private key
    private_key_b64 = root.find(".//privateKey/encodied").text.strip()
    print(f"✓ Found private key ({len(private_key_b64)} chars)")

    # Extract Base64-encoded public key
    public_key_b64 = root.find(".//publicKey/encodied").text.strip()
    print(f"✓ Found public key ({len(public_key_b64)} chars)")

    # Decode private key from Base64
    private_key_der = base64.b64decode(private_key_b64)

    # Convert to PEM format
    private_key_pem = b"-----BEGIN RSA PRIVATE KEY-----\n"
    # Split base64 into 64-character lines
    for i in range(0, len(private_key_b64), 64):
        private_key_pem += (private_key_b64[i : i + 64] + "\n").encode()
    private_key_pem += b"-----END RSA PRIVATE KEY-----\n"

    # Convert public key to PEM format
    public_key_pem = b"-----BEGIN PUBLIC KEY-----\n"
    for i in range(0, len(public_key_b64), 64):
        public_key_pem += (public_key_b64[i : i + 64] + "\n").encode()
    public_key_pem += b"-----END PUBLIC KEY-----\n"

    # Create output directory
    output_dir = Path("firmador_certs")
    output_dir.mkdir(exist_ok=True)

    # Save private key
    private_key_file = output_dir / f"{nit}.key"
    with open(private_key_file, "wb") as f:
        f.write(private_key_pem)
    print(f"✓ Created private key: {private_key_file}")

    # Save public key
    public_key_file = output_dir / f"{nit}_public.key"
    with open(public_key_file, "wb") as f:
        f.write(public_key_pem)
    print(f"✓ Created public key: {public_key_file}")

    # Now create a PKCS#12 (.p12) certificate
    # This requires cryptography library
    try:
        from cryptography.hazmat.primitives import serialization
        from cryptography.hazmat.primitives.serialization import pkcs12
        from cryptography.hazmat.backends import default_backend
        from cryptography import x509
        from cryptography.x509.oid import NameOID
        from cryptography.hazmat.primitives import hashes
        import datetime

        # Load the private key
        private_key = serialization.load_der_private_key(
            private_key_der, password=None, backend=default_backend()
        )

        # Create a self-signed certificate
        subject = issuer = x509.Name(
            [
                x509.NameAttribute(NameOID.COUNTRY_NAME, "SV"),
                x509.NameAttribute(NameOID.ORGANIZATION_NAME, "Ministry Certificate"),
                x509.NameAttribute(NameOID.COMMON_NAME, nit),
            ]
        )

        cert = (
            x509.CertificateBuilder()
            .subject_name(subject)
            .issuer_name(issuer)
            .public_key(private_key.public_key())
            .serial_number(x509.random_serial_number())
            .not_valid_before(datetime.datetime.utcnow())
            .not_valid_after(datetime.datetime.utcnow() + datetime.timedelta(days=3650))
            .sign(private_key, hashes.SHA256(), default_backend())
        )

        # Create PKCS#12 without password (as per your XML - no encryption)
        p12 = pkcs12.serialize_key_and_certificates(
            name=nit.encode(),
            key=private_key,
            cert=cert,
            cas=None,
            encryption_algorithm=serialization.NoEncryption(),
        )

        # Save .p12 file
        p12_file = output_dir / f"{nit}.p12"
        with open(p12_file, "wb") as f:
            f.write(p12)
        print(f"✓ Created PKCS#12 certificate: {p12_file}")

        # Save certificate as PEM
        cert_pem_file = output_dir / f"{nit}.crt"
        with open(cert_pem_file, "wb") as f:
            f.write(cert.public_bytes(serialization.Encoding.PEM))
        print(f"✓ Created certificate PEM: {cert_pem_file}")

    except ImportError:
        print("\n⚠️  'cryptography' library not installed.")
        print("   Install it with: pip install cryptography")
        print(
            "   Only PEM keys were created. Install cryptography to create .p12 file."
        )
        return nit, None

    return nit, p12_file


def create_docker_setup(nit, cert_file):
    """Create docker folder structure and instructions"""

    print("\n" + "=" * 60)
    print("📦 DOCKER FIRMADOR SETUP INSTRUCTIONS")
    print("=" * 60)

    if cert_file:
        print(f"\n1. Copy the certificate to docker/temp folder:")
        print(f"   cp {cert_file} docker/temp/{nit}.p12")
        print(f"\n   OR if using .crt format:")
        print(f"   cp firmador_certs/{nit}.crt docker/temp/{nit}.crt")
    else:
        print(f"\n1. Install cryptography library first:")
        print(f"   pip install cryptography")
        print(f"   Then run this script again.")

    print(f"\n2. Your docker folder structure should look like:")
    print(
        f"""
    docker/
    ├── docker-compose.yml
    ├── svfe-api.env
    └── temp/
        └── {nit}.p12    ← Your certificate here (JUST THE NIT AS FILENAME)
    """
    )

    print(f"\n3. Configure svfe-api.env (for non-SSL):")
    print(
        f"""
    # Remove or comment out this line for non-SSL:
    # SPRING_PROFILES_ACTIVE=ssl
    
    # No other configuration needed for non-SSL
    """
    )

    print(f"\n4. Start the firmador service:")
    print(f"   cd docker")
    print(f"   docker-compose up -d")

    print(f"\n5. Test the service:")
    print(f"   curl http://localhost:8113/firmardocumento/status")

    print(f"\n6. When calling the signing API, use:")
    print(
        f"""
    {{
      "nit": "{nit}",
      "activo": true,
      "passwordPri": "",     ← Empty string (no password)
      "dteJson": "{{...}}"
    }}
    """
    )

    print("\n" + "=" * 60)


def main():
    if len(sys.argv) != 2:
        print("Usage: python3 prepare_cert_for_firmador.py <xml_cert_file>")
        print("Example: python3 prepare_cert_for_firmador.py 06143005061013.crt")
        sys.exit(1)

    xml_file = sys.argv[1]

    if not os.path.exists(xml_file):
        print(f"❌ Error: File '{xml_file}' not found")
        sys.exit(1)

    try:
        nit, cert_file = extract_cert_from_xml(xml_file)
        create_docker_setup(nit, cert_file)

        print("\n✅ Certificate extraction completed successfully!")
        print(f"📁 All files saved in: firmador_certs/")

    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback

        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
