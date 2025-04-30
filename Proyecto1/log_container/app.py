from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import json
import matplotlib.pyplot as plt
from datetime import datetime

app = FastAPI()
LOG_FILE = "/app/logs/logs.json"

class LogEntry(BaseModel):
    container_id: str
    category: str
    created_at: str
    deleted_at: str | None

@app.post("/logs")
async def add_logs(logs: list[LogEntry]):
    try:
        with open(LOG_FILE, "a") as f:
            json.dump([log.dict() for log in logs], f)
            f.write("\n")
        return {"status": "success"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/generate_graphs")
async def generate_graphs():
    logs = []
    with open(LOG_FILE, "r") as f:
        for line in f:
            logs.extend(json.loads(line))

    # Gráfica 1: Número de contenedores por categoría
    categories = {"ram": 0, "cpu": 0, "io": 0, "disk": 0}
    for log in logs:
        categories[log["category"]] += 1
    plt.bar(categories.keys(), categories.values())
    plt.title("Contenedores por Categoría")
    plt.savefig("/app/logs/containers_by_category.png")
    plt.close()

    # Gráfica 2: Tiempo de vida de contenedores
    lifetimes = []
    for log in logs:
        if log["deleted_at"]:
            created = datetime.fromisoformat(log["created_at"].replace("Z", "+00:00"))
            deleted = datetime.fromisoformat(log["deleted_at"].replace("Z", "+00:00"))
            lifetimes.append((deleted - created).total_seconds())
    plt.hist(lifetimes, bins=20)
    plt.title("Distribución de Tiempo de Vida")
    plt.savefig("/app/logs/lifetime_distribution.png")
    plt.close()

    return {"status": "graphs generated"}