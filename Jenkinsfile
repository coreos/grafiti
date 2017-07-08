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

node('worker && ec2') {

  def builder_image = "quay.io/coreos/grafiti"

  stage('Build') {
    checkout scm
    sh """#!/bin/bash -ex
    docker build -t "$builder_image" .
    """
  }
  stage('Test') {
    withCredentials(creds) {
      withDockerContainer(builder_image) {
        sh """#!/bin/bash -ex
        cd /go/src/github.com/coreos/grafiti
        make test
        """
      }
    }
  }
  stage('Push') {
    if (env.BRANCH_NAME == 'master') {
      withCredentials(quay_creds) {
        sh """#!/bin/bash -ex
        docker login -u="$QUAY_USERNAME" -p="$QUAY_PASSWORD" quay.io
        docker push "$builder_image"
        """
      }
    }
  }
}
