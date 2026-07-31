# Objdiff Summary

`tests/diffs-verbose/left.yml` <-> `tests/diffs-verbose/right.yml`

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| 1 | 2 | 1 |
## change `apps/v1>Deployment>>nginx-deployment`

<details><summary>View Diff</summary>

``` diff
--- tests/diffs-verbose/left.yml apps/v1>Deployment>>nginx-deployment
+++ tests/diffs-verbose/right.yml apps/v1>Deployment>>nginx-deployment
@@ -5,7 +5,7 @@
   labels:
     app: nginx
 spec:
-  replicas: 1
+  replicas: 3
   selector:
     matchLabels:
       app: nginx
@@ -16,6 +16,6 @@
     spec:
       containers:
       - name: nginx
-        image: nginx:1.14.3
+        image: nginx:1.14.2
         ports:
         - containerPort: 80
```

</details>

## change `v1>Pod>default>nginx-common`

<details><summary>View Diff</summary>

``` diff
--- tests/diffs-verbose/left.yml v1>Pod>default>nginx-common
+++ tests/diffs-verbose/right.yml v1>Pod>default>nginx-common
@@ -8,4 +8,4 @@
   - name: nginx
     image: nginx:1.14.2
     ports:
-    - containerPort: 80
+    - containerPort: 81
```

</details>

## destroy `v1>Pod>default>nginx-left`

<details><summary>View Diff</summary>

``` diff
--- tests/diffs-verbose/left.yml v1>Pod>default>nginx-left
+++ tests/diffs-verbose/right.yml v1>Pod>default>nginx-left
@@ -1,11 +0,0 @@
-apiVersion: v1
-kind: Pod
-metadata:
-  name: nginx-left
-  namespace: default
-spec:
-  containers:
-  - name: nginx
-    image: nginx:1.14.2
-    ports:
-    - containerPort: 80
```

</details>

## add `v1>Pod>default>nginx-right`

<details><summary>View Diff</summary>

``` diff
--- tests/diffs-verbose/left.yml v1>Pod>default>nginx-right
+++ tests/diffs-verbose/right.yml v1>Pod>default>nginx-right
@@ -0,0 +1,11 @@
+apiVersion: v1
+kind: Pod
+metadata:
+  name: nginx-right
+  namespace: default
+spec:
+  containers:
+  - name: nginx
+    image: nginx:1.14.2
+    ports:
+    - containerPort: 80
```

</details>
