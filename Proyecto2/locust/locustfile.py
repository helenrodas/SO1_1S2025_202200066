from locust import HttpUser, task, between
import json

class TweetUser(HttpUser):
    wait_time = between(1, 5)

    @task
    def post_tweet(self):
        tweet = {
            "description": "Está lloviendo",
            "country": "GT",
            "weather": "Lluvioso"
        }
        self.client.post("/input", json=tweet, headers={"Content-Type": "application/json"})