# Objdiff Summary: tests/left-only/left.yml <-> tests/left-only/right.yml

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 0 | 1 |

<details>
<summary><b>destroy</b> <code>v1>Pod>default>nginx</code></summary>

``` diff
--- tests/left-only/left.yml v1>Pod>default>nginx
+++ tests/left-only/right.yml v1>Pod>default>nginx
@@ -1,11 +0,0 @@
-apiVersion: v1
-kind: Pod
-metadata:
-  name: nginx
-  namespace: default
-spec:
-  containers:
-  - name: nginx
-    image: nginx:1.14.2
-    ports:
-    - containerPort: 80
```

</details>
