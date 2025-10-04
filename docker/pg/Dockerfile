FROM postgres:18

ENV PGDATA /var/lib/postgresql/data

RUN apt-get update && \
    apt-get install --yes --no-install-recommends --no-install-suggests \
    postgresql-10 \
    postgresql-contrib \
    gnupg \
    gpg-agent \
    time \
    bc \
    jq \
    s3cmd \
    python3-setuptools \
    python3-pip \
    lzop \
    pv

COPY docker/pg/PGP_KEY /tmp/PGP_KEY
COPY docker/pg/SSH_KEY /tmp/SSH_KEY
COPY docker/pg/gpg.conf /var/lib/postgresql/.gnupg/gpg.conf
COPY docker/pg/gpg-agent.conf /var/lib/postgresql/.gnupg/gpg-agent.conf

COPY docker/common/s3cfg /var/lib/postgresql/.s3cfg

COPY docker/pg/walg.json /tmp/walg.json

RUN rm -rf $PGDATA

RUN chmod 777 /var/lib/postgresql/.gnupg
