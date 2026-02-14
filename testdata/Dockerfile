FROM golang:1.25-alpine

RUN apk add --no-cache bash zsh fish git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o kubert main.go

CMD ["go", "test", "-run", "TestCreateTempKubeconfigFile_Isolation|TestLaunchShells", "./cmd", "-v"]
