# Intentionally vulnerable file for integration testing.
import os
import requests
import subprocess
from flask import Flask, request

app = Flask(__name__)

# SEC-305: Debug mode enabled
app.run(debug=True)

# SEC-101: Hardcoded API key
api_key = "sk_real_hardcoded_key_1234567890abcdef"

# SEC-302: SSL verification disabled
response = requests.get("https://api.example.com", verify=False)

# SEC-203: Command injection via os.system
user_input = request.args.get("cmd")
os.system(user_input)

# SEC-203: shell=True
subprocess.run(user_input, shell=True)

# SEC-201: SQL string concatenation (f-string)
user_id = request.args.get("id")
query = f"SELECT * FROM users WHERE id = {user_id}"

# TODO: add authentication before shipping (SEC-501)
