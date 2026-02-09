FROM postgres:16-alpine

LABEL maintainer="synapse"
LABEL description="PostgreSQL database for Synapse Event Network"

# Default superuser password (used by the postgres entrypoint)
ENV POSTGRES_PASSWORD=postgres

# Copy the initialization SQL and shell script into the
# docker-entrypoint-initdb.d directory. The postgres image
# runs these scripts (in filename order) only when the data
# directory is empty — i.e. on first container start.
# Prefixing with 01/02 ensures ordering: shell script first,
# then SQL as a fallback if run standalone.
COPY init-db.sh  /docker-entrypoint-initdb.d/01-init-db.sh
COPY init.sql    /docker-entrypoint-initdb.d/init.sql

RUN chmod +x /docker-entrypoint-initdb.d/01-init-db.sh

EXPOSE 5432
