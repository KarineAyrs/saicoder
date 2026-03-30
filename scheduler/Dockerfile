FROM python:3.11-slim

WORKDIR /bin/scheduler

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8080 8081

ENTRYPOINT ["python", "main.py"]
CMD ["-c", "config.yaml"]
