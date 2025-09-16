# nodeprep-controller

Checks Kubernetes nodes for a `spectrocloud.com/nodeprep` label and if the label has a value of "completed", removes the `spectrocloud.com/nodeprep:NoSchedule` taint from that node. Useful for preventing workloads from landing onto a node before all the prereq actions for the node are completed.
