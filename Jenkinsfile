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

  def grafiti_version = sh(returnStdout: true, script: "./scripts/git-version").trim()

  stage('Test') {
    withCredentials(creds) {
      sh """#!/bin/bash -ex
      . ./scripts/docker-test
      """
    }
  }
  stage('Push') {
    if (env.BRANCH_NAME == 'master') {
      withCredentials(quay_creds) {
        sh """#!/bin/bash -ex
        docker login -u="$QUAY_ROBOT_USERNAME" -p="$QUAY_ROBOT_SECRET" quay.io
        make docker-image
        docker push quay.io/coreos/grafiti:$grafiti_version
        """
      }
    }
  }
}
