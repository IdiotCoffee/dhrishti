import time

import requests

SERVER_URL = "http://server:8080"

while True:
    response = requests.get(SERVER_URL)
    print("Response: ", response.text)
    time.sleep(3)
