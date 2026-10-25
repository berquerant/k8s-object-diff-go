# Objdiff Summary: tests/diff-ignore-matching-lines-env/left.yml <-> tests/diff-ignore-matching-lines-env/right.yml

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |

<details>
<summary><b>change</b> <code>apps/v1>Deployment>>nginx-deployment</code></summary>

``` diff
--- tests/diff-ignore-matching-lines-env/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-ignore-matching-lines-env/right.yml apps/v1>Deployment>>nginx-deployment
@@ -5,7 +5,7 @@
   labels:
     app: nginx
 spec:
-  replicas: 3
+  replicas: 1
   selector:
     matchLabels:
       app: nginx
```

</details>
