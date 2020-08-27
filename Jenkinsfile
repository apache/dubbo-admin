@Library('visenze-lib')_

pipeline {
  // choose a suitable agent to build
  agent {
    label 'build'
  }

  options {
    timestamps()
  }

  parameters {
    string(name: 'DOCKER_REGISTRY', defaultValue: '', description: 'leave empty to use dockerhub as registry')
    string(name: 'DOCKER_REGISTRY_CREDENTIAL', defaultValue: 'docker-hub-credential',
        description: 'The credential ID stored in Jenkins to pull/push to docker registry')
    string(name: 'AWS_MAVEN_CREDENTIALS_ID', defaultValue: 'visenze-test-maven-repo')
  }

  tools {
    maven 'Default' // enable maven
  }

  stages {
    stage('Checkout') {
      steps {
        // checkout the code from scm (github)
        checkout scm
      }
    }

    stage('Package') {
      steps {
        script {
          sh "mvn clean compile package install"
        }
      }
    }

    stage('Docker Build') {
      when {
        expression {
          return doBuildAndTest(env.BRANCH_NAME)
        }
      }
      steps {
        script {
          // assume the Dockerfile is directly under WORKSPACE
          def commitHash = getCommit()
          // get jar build version for cas-server (version inherit from cas-parent)
          def jarVersion = readMavenPom(file: 'pom.xml').version
          echo "jarVersion ${jarVersion}"
          // pull build image
          docker.withRegistry(params.DOCKER_REGISTRY, params.DOCKER_REGISTRY_CREDENTIAL) {
            // assume the docker registry is visenze/dubbo-admin
            docker.build("visenze/dubbo-admin:${commitHash}", "-f docker/Dockerfile .")
          }
        }
      }
    }

    stage('Docker Push') {
      when {
        expression {
          return doBuildAndTest(env.BRANCH_NAME)
        }
      }
      steps {
        script {
          def commitHash = getCommit()
          def version = getVersion()
          docker.withRegistry(params.DOCKER_REGISTRY, params.DOCKER_REGISTRY_CREDENTIAL) {
            def image = docker.image("visenze/dubbo-admin:${commitHash}")
            // push all the tags
            def tags = genDockerTags(env.BRANCH_NAME, commitHash, version, env.BUILD_NUMBER)
            // keep the last tag as a global variable used by later stages
            TAG = tags[-1]
            tags.add(commitHash)

            tags.each {
              retry(2) {
                image.push(it)
              }
            }
          }
        }
      }
    }


    stage('Archive') {
      when {
        expression {
          return doBuildAndTest(env.BRANCH_NAME) || isProdRelease(env.BRANCH_NAME)
        }
      }
      steps {
        script {
          def archive = [
            docker_tag: TAG,
            version: getVersion(),
            docker_repos: ['visenze/dubbo-admin']
          ]
          writeFile(file: "${WORKSPACE}/version.json", text: groovy.json.JsonOutput.toJson(archive))
          archiveArtifacts(artifacts: "version.json", allowEmptyArchive: true)
        }
      }
    }
  }

}



// Get commit sha
def getCommit() {
  return sh(
    script: "(cd '${WORKSPACE}'; git rev-parse HEAD)", returnStdout: true
  ).trim()
}

def getVersion() {
  // read version from pom
  return readMavenPom(file: 'pom.xml').version
}

// Get snapshot version
def genSnapShotVersion(version) {
  def suffix = "-SNAPSHOT"
    if (!version.endsWith(suffix)) {
      return version + suffix
    }
    return version
}

// Get Release version
def genReleaseVersion(version) {
  def suffix = "-SNAPSHOT"
  if (version.endsWith(suffix)) {
    return version.substring(0, version.length() - suffix.length())
  }
  return version
}

def genDockerTags(branch, commit, version=null, buildNumber=null) {
  def tags = []
  def shortCommit = commit.substring(0, 9)
  tags.add(branch.replaceAll("/", "_"))
  tags.add("${branch.replaceAll("/", "_")}-${shortCommit}")
  //if(branch == 'production') {
  //  assert version !=null
  //  tags.add("${version}")
  //} else {
  //  assert version != null && buildNumber != null
  //  tags.add("${version}.${buildNumber}-${shortCommit}")
  //}
  return tags
}

def isPullRequest(branch) {
  return branch.startsWith('PR')
}

def doBuildAndTest(branch) {
  return branch.startsWith('PR') || branch == 'staging' || branch == 'production'
}

def isProdRelease(branch) {
  return env.BRANCH_NAME == "production"
}
