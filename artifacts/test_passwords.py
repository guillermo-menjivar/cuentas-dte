# The hash from your XML certificate
target_hash = "0f43fa5142a28cab25c365e5a5d48c457bf8896740907745d11fb54a74bdcb3d2598ac35dd365c40a194a7a31007d443c0e7dc3ff560d236754b7fecc4337f9e"

# Possible passwords to test
passwords = [
    "Paredes30050606",
    "paredes30050606",
    "PAREDES30050606",
    "Paredes@30050606",
    "06143005061013",  # Your NIT
    "1726313",  # Your NRC
    # Add any other passwords you might have used
]

print("Testing passwords...\n")

for pwd in passwords:
    # Hash with SHA-512 (same as the firmador does)
    hashed = hashlib.sha512(pwd.encode("utf-8")).hexdigest()

    print(f"Password: {pwd}")
    print(f"Hash:     {hashed}")

    if hashed == target_hash:
        print(f"✅ MATCH! Password is: {pwd}\n")
        break
    else:
        print(f"❌ No match\n")
else:
    print("❌ None of the passwords matched!")
    print(
        "\nYou need to find the original password used when generating the certificate."
    )
