FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY ml-core /ml-core
COPY analyser-ml /app

RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "main.py"]
