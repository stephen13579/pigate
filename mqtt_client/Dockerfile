FROM python:3.9-slim

WORKDIR /app

COPY requirements.txt requirements.txt
RUN pip install -r requirements.txt

COPY client/ /app/client/
COPY tests/ /app/tests/

# CMD ["python", "-m", "client.mqtt_client"]
