#!/usr/bin/env sh

# 0m
echo 1
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 1
sleep 120
# 2m
echo 2
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 2
sleep 120
# 4m
echo 3
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 3
sleep 120
# 6m
echo 4
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 4
sleep 120
# 8m
echo 5
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 5
sleep 120
# 10m
echo 6
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 6
sleep 120
# 12m
echo 7
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 7
sleep 120
# 14m
echo 8
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 8
sleep 120
# 16m
echo 9
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 9
sleep 120
# 18m
echo 10
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 10
sleep 120
# 20m
echo "delete thanos-receive-default-1"
kubectl delete pod -n thanos thanos-receive-default-1
sleep 300
# 25m
echo "delete thanos-receive-default-6"
kubectl delete pod -n thanos thanos-receive-default-6
sleep 300
# 30m
echo 1
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 1
sleep 300
# 35m
echo 0
kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 0
echo "done"
exit 0
