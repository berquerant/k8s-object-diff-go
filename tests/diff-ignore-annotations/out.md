# Objdiff Summary

`tests/diff-ignore-annotations/left.yml` <-> `tests/diff-ignore-annotations/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diff-ignore-annotations/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-ignore-annotations/right.yml apps/v1>Deployment>>nginx-deployment
@@ -4,7 +4,8 @@
   name: nginx-deployment
   labels:
     app: nginx
-  annotations: {}
+  annotations:
+    anotherAnnotation: right
 spec:
   replicas: 3
   selector:
```

</details>
