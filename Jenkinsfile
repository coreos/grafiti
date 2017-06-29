def creds = [
  usernamePassword(
    credentialsId: 'jenkins-tectonic-installer',
    usernameVariable: 'AWS_ACCESS_KEY_ID',
    passwordVariable: 'AWS_SECRET_ACCESS_KEY'
  )
]

def quay_creds = [
  usernamePassword(
    credentialsId: 'quay-robot',
    passwordVariable: 'QUAY_ROBOT_SECRET',
    usernameVariable: 'QUAY_ROBOT_USERNAME'
  )
]

pipeline {
  agent none
  environment {
    GRAFITI_IMAGE = 'quay.io/coreos/grafiti'
  }
  stages {
    stage('Build & Test') {
      steps {
        node('worker && ec2') {
          withCredentials(creds) {
            checkout scm
            sh """#!/bin/bash -ex
            docker build -t "$GRAFITI_IMAGE" .
            docker run --rm \
          	 -e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
           	 -e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
             "$GRAFITI_IMAGE" ash -c 'cd \$GRAFITI_ABS_PATH && make test'
            """
          }
        }
      }
    }
    stage('Push') {
      when {
        branch 'master'
      }
      steps {
        node('worker && ec2') {
          withCredentials(quay_creds) {
            sh """#!/bin/bash -ex
            docker login -u="$QUAY_USERNAME" -p="$QUAY_PASSWORD" quay.io
            docker build -t "$GRAFITI_IMAGE" .
            docker push "$GRAFITI_IMAGE"
            """
          }
        }
      }
    }
  }
}
