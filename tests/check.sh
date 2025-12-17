cd /mnt

make build
cp kubsh /usr/local/bin/

cd /opt
pytest -v --log-cli-level=10 .