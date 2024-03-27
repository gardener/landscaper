<%
  with open(kubeconfig_path) as f:
    lines = f.readlines()

  lines = map(str.rstrip, lines)
%>
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: ${targetName}
  namespace: ${namespace}
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
% for line in lines:
      ${line}
% endfor