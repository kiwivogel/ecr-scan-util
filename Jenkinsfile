library('jenkins-shared-library')

pipeline {

    environment {
        def COMPONENT_NAME = "esa"
        def GO111MODULE = 'on'
        def GOARCH = "amd64"
        def GOOS = "linux"
        def TARGET = """${sh(
                        returnStdout: true,
                        script: 'echo ${COMPONENT_NAME}_${GOOS}_${GOARCH}'
                      )}"""
    }

    options {
        ansiColor('xterm')
        timestamps()
        buildDiscarder(logRotator(numToKeepStr: '15'))
        disableConcurrentBuilds()
    }
    agent { label Agent.golang }
    tools {
        go "Go 1.13.5"
    }
    stages {
        stage('Go Build'){
            steps {
                set_build_version()
                sh '''#!/bin/bash
                    set -e
                    echo "Getting modules"
                    go mod download
                    echo "Compiling..."
                    go build -o ${TARGET}
                    echo "Done"
                    chmod +x ${TARGET}
                    '''
            }
        }
        stage('Go Run') {
            steps {
                with_ecr_credentials {
                    sh '''#!/bin/bash
                        echo "Running ${TARGET}..."
                        ./${TARGET}
                        '''
                }
            }
        }
    }
    post {
        always {
            junit 'reports/*.xml'
            default_post_actions()
            wrap([$class: 'MesosSingleUseSlave']) {}
        }
    }
}