FROM ubuntu:18.04

COPY . .
RUN apt-get update -y
RUN apt-get install net-tools -y
RUN apt-get install default-jdk -y
RUN apt-get install wget -y
RUN apt-get install curl -y
RUN wget http://apache.spd.co.il/kafka/2.5.0/kafka_2.13-2.5.0.tgz
RUN tar xzvf kafka_2.13-2.5.0.tgz
