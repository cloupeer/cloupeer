package k8s

import (
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"cloupeer.io/cloupeer/internal/cloudhub/core/model"
	iovv1alpha2 "cloupeer.io/cloupeer/pkg/apis/iov/v1alpha2"
)

// ToModel converts a K8s CRD object to a Core Model entity.
func ToModel(crd *iovv1alpha2.Vehicle) *model.Vehicle {
	return &model.Vehicle{
		ID:                crd.Name, // Map K8s Name to Model ID
		ReportedVersion:   crd.Status.Profile.Firmware.Version,
		Online:            crd.Status.Online,
		LastHeartbeatTime: extractTime(crd.Status.LastHeartbeatTime),
		DesiredVersion:    crd.Spec.Profile.Firmware.Version,
	}
}

// ToCRD converts a Core Model entity to a K8s CRD object (for creation).
func ToCRD(ns string, v *model.Vehicle) *iovv1alpha2.Vehicle {
	return &iovv1alpha2.Vehicle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v.ID,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":    "cpeer-hub",
				"iov.cloupeer.io/auto-discovered": strconv.FormatBool(v.IsRegister),
			},
		},
		Spec: iovv1alpha2.VehicleSpec{
			VIN: "",
			Profile: iovv1alpha2.VehicleProfile{
				Firmware: iovv1alpha2.FirmwareConfig{
					Version: v.DesiredVersion,
				},
			},
		},
		Status: iovv1alpha2.VehicleStatus{
			Online: v.Online,
			Profile: iovv1alpha2.VehicleProfile{
				Firmware: iovv1alpha2.FirmwareConfig{
					Version: v.ReportedVersion,
				},
			},
			LastHeartbeatTime: toMetaTime(v.LastHeartbeatTime),
		},
	}
}

// Helper functions for time conversion
func extractTime(t *metav1.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time
}

func toMetaTime(t time.Time) *metav1.Time {
	if t.IsZero() {
		return nil
	}
	mt := metav1.NewTime(t)
	return &mt
}
