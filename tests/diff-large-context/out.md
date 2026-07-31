# Objdiff Summary

`tests/diff-large-context/left.yml` <-> `tests/diff-large-context/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diff-large-context/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff-large-context/right.yml apps/v1>Deployment>>nginx-deployment
@@ -1,21 +1,21 @@
 apiVersion: apps/v1
 kind: Deployment
 metadata:
   name: nginx-deployment
   labels:
     app: nginx
 spec:
-  replicas: 3
+  replicas: 1
   selector:
     matchLabels:
       app: nginx
   template:
     metadata:
       labels:
         app: nginx
     spec:
       containers:
       - name: nginx
-        image: nginx:1.14.2
+        image: nginx:1.14.3
         ports:
         - containerPort: 80
```

</details>
