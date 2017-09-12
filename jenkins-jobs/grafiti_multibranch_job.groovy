#!/bin/env groovy​

multibranchPipelineJob("grafiti") {
  description("Tag and remove AWS Resources with Automation.\nThis job is managed by grafiti git repository.\nChanges here will be reverted automatically.")
  branchSources {
    github {
      scanCredentialsId("37477e0c-2ab6-46fe-a83b-64b1add4777d")
      checkoutCredentialsId("37477e0c-2ab6-46fe-a83b-64b1add4777d")
      apiUri("")
      repoOwner("coreos")
      repository("grafiti")
      buildForkPRHead(false)
      buildForkPRMerge(true)
      buildOriginBranch(true)
      buildOriginBranchWithPR(false)
      buildOriginPRHead(false)
      buildOriginPRMerge(true)
    }
  }

  configure { project ->
    project / 'sources' / 'data' / 'jenkins.branch.BranchSource'/ strategy(class: 'jenkins.branch.DefaultBranchPropertyStrategy') {
      properties(class: 'java.util.Arrays$ArrayList') {
        a(class: 'jenkins.branch.BranchProperty-array'){
          'jenkins.branch.NoTriggerBranchProperty'()
        }
      }
    }
  }

  orphanedItemStrategy {
    discardOldItems {
      numToKeep(50)
    }
  }
  triggers {
    periodic(10080)
  }
}
