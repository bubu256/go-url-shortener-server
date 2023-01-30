FROM postgres:15.1-alpine

ENV POSTGRES_USER=postgres
# ENV POSTGRES_PASSWORD=SetYouPassword # set here or in docker run
ENV POSTGRES_DB postgres
COPY initDB.sql /docker-entrypoint-initdb.d/

EXPOSE 5432 

# use for run:
# docker build -t short_db .
# docker run -d --rm --name pg_short -p 5432:5432 -v pg_shortdata:/var/lib/postgresql/data -e POSTGRES_PASSWORD=SetYouPassword short_db