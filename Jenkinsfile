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

  checkout scm
  
  def git_branch = env.BRANCH_NAME
  def git_commit = sh(returnStdout: true, script: "git rev-parse --short HEAD").trim()
  def builder_image = "quay.io/coreos/grafiti:${git_commit}"

  stage('Build') {
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
    if (git_branch == 'master') {
      withCredentials(quay_creds) {
        sh """#!/bin/bash -ex
        docker login -u="$QUAY_USERNAME" -p="$QUAY_PASSWORD" quay.io
        docker push "$builder_image"
        """
      }
    }
  }
}
