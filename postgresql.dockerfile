FROM postgres:12.2
ENV POSTGRES_PASSWORD password
COPY sql/* /tmp/
COPY scripts/init-user-db.sh /docker-entrypoint-initdb.d/
