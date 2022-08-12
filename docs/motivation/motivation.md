# What is Landscaper?

The Landscaper provides means to describe, install and maintain cloud-native landscapes. It allows to express an order of building blocks, connect output with input data and ultimately, bring your landscape to life.

What does a 'landscape' consist of? In this context, it refers not only to application bundles, but also includes infrastructure components in public, hybrid and private environments.

While tools like Terraform, Helm or native Kubernetes resources work well in their specific problem space, until now it has always been a manual task to connect these tools / make them work together. Landscaper solves this problem and offers a fully-automated installation flow, even between the mentioned tools. In order to do so, it translates so-called "blueprints" of components into actionable items and instructs (or orchestrates) tools like Helm or Terraform to deploy the individual items. In turn, the produced output of one item can be used as input for a subsequent step - regardless of the tool used underneath. Since implemented as a set of Kubernetes operators, Landscaper uses the concept of reconciliation to enforce a desired state, which also allows for updates to be rolled out smoothly.
