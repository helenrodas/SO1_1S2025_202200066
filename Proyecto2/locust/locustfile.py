import random
from locust import HttpUser, task, between

# Lista de datos para generar objetos JSON
descriptions = ["Está lloviendo", "Cielo despejado", "Está nublado", "Llovizna ligera"]
countries = ["GT", "MX", "US", "BR", "AR"]
weathers = ["Lluvioso", "Nublado", "Soleado"]

# Generar 10,000 objetos JSON
tweets = [
    {
        "description": random.choice(descriptions),
        "country": random.choice(countries),
        "weather": random.choice(weathers)
    }
    for _ in range(10000)
]

class TweetUser(HttpUser):
    wait_time = between(1, 3)  # Tiempo de espera entre tareas (en segundos)

    @task
    def send_tweet(self):
        # Seleccionar un tweet aleatorio de la lista
        tweet = random.choice(tweets)
        headers = {"Content-Type": "application/json"}
        # Enviar la petición POST al Ingress
        self.client.post("/input", json=tweet, headers=headers)