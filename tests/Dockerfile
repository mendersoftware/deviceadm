FROM ubuntu:14.04

COPY / /testing
RUN apt-get -y -qq update && apt-get -qq -y install python-dev python-pip libffi-dev libssl-dev
RUN pip2 install --quiet pytest bravado pycrypto
WORKDIR /testing
RUN find . -iname "*.pyc" -delete
ENTRYPOINT ["bash", "run.sh"]
