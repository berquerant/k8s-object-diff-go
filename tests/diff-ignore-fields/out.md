# Objdiff Summary

`tests/diff-ignore-fields/left.yml` <-> `tests/diff-ignore-fields/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diff-ignore-fields/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-ignore-fields/right.yml apps/v1>Deployment>>nginx-deployment
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
