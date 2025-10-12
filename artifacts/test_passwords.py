import hashlib

password = "Paredes30050606"
hashed = hashlib.sha512(password.encode("utf-8")).hexdigest()
print(hashed)
