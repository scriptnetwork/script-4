FROM golang:latest
ENV GOPATH=/app
ENV PATH=$GOPATH/bin:$PATH
WORKDIR /app/src/github.com/scripttoken/script
COPY . .
RUN make install
RUN cp -r ./integration/testnet_amber ../
EXPOSE 28888
CMD script start --config=../testnet_amber/node --password="qwertyuiop"

