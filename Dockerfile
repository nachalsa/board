# 빌드 스테이지
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Go 모듈 파일 복사
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 애플리케이션 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# 실행 스테이지
FROM alpine:latest

# 필요한 패키지 설치
RUN apk --no-cache add ca-certificates tzdata

# 빌드 인수로 UID/GID 받기
ARG UID=1000
ARG GID=1000

# 사용자 생성 (빌드 인수 사용)
RUN addgroup -g ${GID} appuser && \
    adduser -D -u ${UID} -G appuser appuser

# 타임존 설정
ENV TZ=Asia/Seoul

# 작업 디렉토리 설정
WORKDIR /app

# 빌드된 바이너리 복사
COPY --from=builder /app/main .

# 템플릿과 정적 파일 복사
COPY web/ ./web/

# 업로드 디렉토리 생성 및 권한 설정 (루트 권한으로)
RUN mkdir -p files && \
    chown -R ${UID}:${GID} /app

# 사용자 전환 (마지막에)
USER ${UID}:${GID}

# 포트 노출
EXPOSE 80 8081

# 애플리케이션 실행
CMD ["./main"]
