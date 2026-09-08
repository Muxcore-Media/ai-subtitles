package internal

import (
	"context"
	"encoding/json"
	"fmt"

	meshv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/mesh/v1"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"google.golang.org/grpc"
)

type aiMeshServer struct {
	meshv1.UnimplementedModuleMeshServer
	moduleID string
	settings modulesdk.SettingsHandler
	h        meshSettings
}

func registerAIMesh(srv *grpc.Server, moduleID string, h meshSettings) {
	meshv1.RegisterModuleMeshServer(srv, &aiMeshServer{
		moduleID: moduleID,
		settings: modulesdk.SettingsHandlerFromProvider(h),
		h:        h,
	})
}

func (s *aiMeshServer) Call(ctx context.Context, req *meshv1.CallRequest) (*meshv1.CallResponse, error) {
	if req.GetTargetModule() != "" && req.GetTargetModule() != s.moduleID {
		return &meshv1.CallResponse{Error: fmt.Sprintf("wrong target module %q", req.GetTargetModule())}, nil
	}
	switch req.GetMethod() {
	case "Settings":
		raw, err := json.Marshal(s.settings.List())
		if err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		return &meshv1.CallResponse{Payload: raw}, nil
	case "UpdateSetting":
		var body struct{ Key, Value string }
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		if err := s.settings.Update(body.Key, body.Value); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		return &meshv1.CallResponse{Payload: []byte(`{"ok":true}`)}, nil
	default:
		raw, err := s.h.handleMesh(ctx, req.GetMethod(), req.GetPayload())
		if err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		return &meshv1.CallResponse{Payload: raw}, nil
	}
}

func (s *aiMeshServer) StreamCall(meshv1.ModuleMesh_StreamCallServer) error {
	return fmt.Errorf("StreamCall not supported")
}

func fmtError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
