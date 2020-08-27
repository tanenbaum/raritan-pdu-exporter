package main

import "k8s.io/klog"

func main() {
	klog.InitFlags(nil)

	Execute()
}
