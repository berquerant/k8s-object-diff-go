# Objdiff Summary: tests/diff-ignore-labels/left.yml <-> tests/diff-ignore-labels/right.yml

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |

<details>
<summary><b>change</b> <code>apps/v1>Deployment>>nginx-deployment</code></summary>

``` diff
--- tests/diff-ignore-labels/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-ignore-labels/right.yml apps/v1>Deployment>>nginx-deployment
@@ -5,7 +5,7 @@
   labels:
     app: nginx
 spec:
-  replicas: 1
+  replicas: 3
   selector:
     matchLabels:
       app: nginx
```

</details>
