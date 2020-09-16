FROM golang:1.13.6-stretch AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=1

WORKDIR /build

# Let's cache modules retrieval - those don't change so often
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code necessary to build the application
# You may want to change this to copy only what you actually need.
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o ecr-scan-util

# Let's create a /dist folder containing just the files necessary for runtime.
# Later, it will be copied as the / (root) of the output image.
WORKDIR /dist
RUN cp /build/ecr-scan-util ./ecr-scan-util

# Copy or create other directories/files your app needs during runtime.
# E.g. this example uses /data as a working directory that would probably
#      be bound to a perstistent dir when running the container normally
RUN mkdir /data

# Create the minimal runtime image, We're using jarlefosen's image because this contains neccesary SSL root certs.
FROM  alpine

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group && \
    echo 'jenkins:x:1000:1000:jenkins:/:' > /user/passwd && \
    echo 'jenkins:x:jenkins:' > /user/group && \
    apk add --no-cache ca-certificates tzdata

COPY --chown=0:0 --from=builder /dist /

# Set up the app to run as a non-root user inside the /data folder
# User ID 65534 is usually user 'nobody'.
# The executor of this image should still specify a user during setup.
COPY --chown=65534:0 --from=builder /data /data
USER 1000
WORKDIR /data

ENTRYPOINT ["/ecr-scan-util"]