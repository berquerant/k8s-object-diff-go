# Objdiff Summary

`tests/diff/left.yml` <-> `tests/diff/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 0 | 1 | 0 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diff/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diff/right.yml apps/v1>Deployment>>nginx-deployment
@@ -5,7 +5,7 @@
   labels:
     app: nginx
 spec:
-  replicas: 3
+  replicas: 1
   selector:
     matchLabels:
       app: nginx
@@ -16,6 +16,6 @@
     spec:
       containers:
       - name: nginx
-        image: nginx:1.14.2
+        image: nginx:1.14.3
         ports:
         - containerPort: 80
```

</details>
