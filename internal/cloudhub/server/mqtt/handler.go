package mqtt

// updateVehicleOnlineStatus updates ONLY the status.online field and lastSeenTime.
// It explicitly DOES NOT touch status.conditions['Ready'] to avoid race conditions with the Controller.
// func (s *HubServer) updateVehicleOnlineStatus(ctx context.Context, vid string, online bool) error {
// 	var vehicle iovv1alpha1.Vehicle
// 	// Use Get + Patch pattern for optimistic locking and minimal payload
// 	if err := s.k8sclient.Get(ctx, types.NamespacedName{Name: vid, Namespace: s.namespace}, &vehicle); err != nil {
// 		return controllerclient.IgnoreNotFound(err)
// 	}

// 	patch := controllerclient.MergeFrom(vehicle.DeepCopy())

// 	// 1. Update the Physical Connectivity State
// 	vehicle.Status.Online = online

// 	// 2. Update Heartbeat
// 	now := metav1.Now()
// 	vehicle.Status.LastSeenTime = &now

// 	// Apply Patch
// 	return s.k8sclient.Status().Patch(ctx, &vehicle, patch)
// }

// handleStatusReport 处理 Agent 上报的状态
// func (s *HubServer) handleStatusReport(ctx context.Context, topic string, payload []byte) {

// 	cmd := &iovv1alpha1.VehicleCommand{}
// 	cmd.Name = statusMsg.CommandName
// 	cmd.Namespace = s.namespace // 假设所有 Command 都在 Hub 所在的 Namespace

// 	// 2. 获取当前对象 (可选，为了更安全的 Patch，或者直接使用 MergePatch)
// 	// 这里我们使用 MergeFrom 进行 Patch
// 	patch := controllerclient.MergeFrom(cmd.DeepCopy())

// 	// 3. 设置新状态
// 	cmd.Status.Phase = iovv1alpha1.CommandPhase(statusMsg.Status)
// 	cmd.Status.Message = statusMsg.Message
// 	now := metav1.Now()
// 	cmd.Status.LastUpdateTime = &now

// 	// 根据状态设置特定的时间戳
// 	if statusMsg.Status == string(iovv1alpha1.CommandPhaseReceived) {
// 		cmd.Status.AcknowledgeTime = &now
// 	} else if statusMsg.Status == string(iovv1alpha1.CommandPhaseSucceeded) ||
// 		statusMsg.Status == string(iovv1alpha1.CommandPhaseFailed) {
// 		cmd.Status.CompletionTime = &now
// 	}

// 	// 4. 执行 Patch
// 	if err := s.k8sclient.Status().Patch(ctx, cmd, patch); err != nil {
// 		log.Error(err, "Failed to patch VehicleCommand status", "name", cmd.Name)
// 		return
// 	}

// 	log.Info("Successfully patched VehicleCommand", "name", cmd.Name, "phase", statusMsg.Status)
// }

// func (s *HubServer) handleOTARequest(ctx context.Context, topic string, payload []byte) {

// 	resp := &pb.OTAResponse{RequestId: req.RequestId}

// 	// 假设固件文件在存储桶中的路径格式为: {version}/vehicle.bin
// 	// 在真实场景中，这里应该查询数据库或 K8s 获取该版本对应的真实 ObjectKey
// 	objectKey := fmt.Sprintf("%s/vehicle.bin", req.DesiredVersion)

// 	// 生成 1 小时有效期的链接
// 	downloadURL, err := s.storage.GeneratePresignedURL(ctx, objectKey, 1*time.Hour)
// 	if err != nil {
// 		log.Error(err, "Failed to generate presigned URL")
// 		resp.ErrorMessage = "Internal Server Error: Storage unavailable"
// 	} else {
// 		resp.DownloadUrl = downloadURL
// 	}

// 	// 发送响应
// 	respBytes, _ := protojson.Marshal(resp)
// 	err = s.mqttclient.Publish(ctx, s.topicbuilder.Build(paths.OTAResponse, req.VehicleId), 1, false, respBytes)
// 	if err != nil {
// 		log.Error(err, "Failed to publish firmware URL response")
// 	} else {
// 		log.Info("Sent Firmware URL", "url", resp.DownloadUrl)
// 	}
// }
