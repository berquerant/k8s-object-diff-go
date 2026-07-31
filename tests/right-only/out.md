# Objdiff Summary

`tests/right-only/left.yml` <-> `tests/right-only/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 1 | 0 | 0 |
## add `v1>Pod>default>nginx`

<details><summary>View Diff</summary>

``` diff
--- tests/right-only/left.yml v1>Pod>default>nginx
+++ tests/right-only/right.yml v1>Pod>default>nginx
@@ -0,0 +1,11 @@
+apiVersion: v1
+kind: Pod
+metadata:
+  name: nginx
+  namespace: default
+spec:
+  containers:
+  - name: nginx
+    image: nginx:1.14.2
+    ports:
+    - containerPort: 80
```

</details>
