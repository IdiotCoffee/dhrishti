FROM python:3.11-slim
WORKDIR /app
COPY server.py .
EXPOSE 8080

CMD ["python", "server.py"]
# CMD means that the server IS the container - if server closes, container shuts down
