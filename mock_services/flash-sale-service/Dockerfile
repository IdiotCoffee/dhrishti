FROM python:3.11-slim

WORKDIR /app

COPY common/ /app/common/
COPY flash-sale-service/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY flash-sale-service/app.py ./app.py

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health')"

CMD ["gunicorn", "--bind", "0.0.0.0:8080", "--workers", "3", "--threads", "6", "--timeout", "90", "app:app"]
