FROM python:3.13-slim

WORKDIR /opt

RUN apt update  \
    && apt install -y --no-install-recommends golang-go libreadline-dev libfuse3-dev \
    && rm -rf /var/lib/apt

COPY tests/requirements.txt .

RUN pip install -r requirements.txt \
  && find / -name "*.pyc" -exec rm -f {} \; \
  && rm -rf /root/.cache/

COPY go.mod go.sum ./
COPY cmd/kubsh/main.go ./

COPY tests/conftest.py tests/test_basic.py tests/test_vfs.py tests/utils.py ./

RUN go build -o kubsh main.go
RUN chmod +x kubsh
RUN cp kubsh /usr/local/bin/

RUN mkdir -p /opt/users
ENV VFS_DIR=/opt/users

CMD ["python", "-m", "pytest", "-v"]