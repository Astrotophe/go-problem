#!groovy

import java.text.SimpleDateFormat
import groovy.json.JsonOutput

@Library('common') _
import com.canalplus.jenkins.tools.Notify
Notify notify = new Notify(this)

def component = "rocket-storagemanager"
def golangVersion = "1.17"
def version = ""
def now = new Date()
def sdf = new SimpleDateFormat("yyyy-MM-dd#HH:mm:ss")
def date = sdf.format(now)
def tgzFile = ""
def current_path = ""
def branchName = ""
def gitRepository = "ssh://git@p1-gaas-git01.cplus:7999/roc/${component}.git"

pipeline {

  agent {
      label 'docker'
  }

  options {
      ansiColor('xterm')
      timeout(time: 1, unit: 'HOURS')
  }

  stages {

    stage('Checkout') {
      steps {
        script {
          notify.push("success", "Starting build #${env.BUILD_NUMBER} for ${component} ${env.BRANCH_NAME}\n${env.BUILD_URL}", "devfactory-ci")
          checkoutGit(gitRepository, env.BRANCH_NAME)
          current_path = sh returnStdout: true, script: "pwd"
          current_path = current_path.replaceAll('\n','')
        }
      }
    }

    stage('Set Version') {
      steps {
        script {
          branchName = env.BRANCH_NAME
          sh "git fetch --tags --all --prune"
          if (env.BRANCH_NAME == 'master') {
              version = input(
                    id: 'version', message: 'Set the Hotfix version', parameters: [
                    [$class: 'StringParameterDefinition', defaultValue: 'X.Y.Z-HFA', description: 'X.Y.Z-HFA', name: 'version']
              ])
              branchName = 'HF'
          } else if (env.BRANCH_NAME == 'release') {
              version = sh returnStdout: true, script: "${current_path}/bin/release"
              version = version.replaceAll('\n','')
          } else {
              version = 'LATEST'
          }
          tgzFile = "${component}-${version}.tgz"
          echo "New version is ${tgzFile}"

          def info = ["name":         "rocket-storagemanager",
                      "version":       version,
                      "gitBranch":     env.BRANCH_NAME,
                      "buildDate":     date,
                      "golangVersion": golangVersion]
          def json = JsonOutput.toJson(info)
          println(json.toString())
          writeFile file: "buildinfo.json", text: json.toString()
        }
      }
    }

    stage('Build') {
      steps {
        script {
          sh """docker pull golang:${golangVersion}; docker run --net=host --dns=8.8.8.8 --privileged --rm -e HTTP_PROXY=http://localhost:3128 -e HTTPS_PROXY=http://localhost:3128 \
                --rm -v ${current_path}:/go/src/rocket-storagemanager \
                -w /go/src/rocket-storagemanager \
                -e GOOS=linux \
                golang:${golangVersion} make build"""
        }
      }
    }

    stage('Publish') {
      steps {
        script {
          sh "tar -czvf ${tgzFile} ${component} buildinfo.json"
          sh "aws s3 cp ${tgzFile} s3://hubmedia-builds/${component}/${branchName}/${tgzFile} --grants full=emailaddress=cedric.rigaud@canal-plus.com read=emailaddress=dsi-prod-cloud@canal-plus.com"
        }
      }
    }

    stage('Push git tag version') {
      when {
        expression {
          env.BRANCH_NAME == 'release'
        }
      }
      steps {
        sh "git tag ${version}"
        sh "git push --tag"
      }
    }

  }

  post {
      success {
          script {
              notify.push("success", "Build #${env.BUILD_NUMBER} : ${component} on branch ${env.BRANCH_NAME} succeeded\n${env.BUILD_URL}", "devfactory-ci")
          }
      }
      unstable {
          script {
              notify.push("unstable", "Build #${env.BUILD_NUMBER} : ${component} on branch ${env.BRANCH_NAME} unstable\n${env.BUILD_URL}", "devfactory-ci")
          }
      }
      failure {
          script {
              notify.push("failure", "Build #${env.BUILD_NUMBER} : ${component} on branch ${env.BRANCH_NAME} failed\n${env.BUILD_URL}", "devfactory-ci")
          }
      }
  }

}
