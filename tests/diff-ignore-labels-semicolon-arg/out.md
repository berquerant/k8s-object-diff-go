# Objdiff Summary

`tests/diff-ignore-labels-semicolon-arg/left.yml` <-> `tests/diff-ignore-labels-semicolon-arg/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diff-ignore-labels-semicolon-arg/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-ignore-labels-semicolon-arg/right.yml apps/v1>Deployment>>nginx-deployment
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
