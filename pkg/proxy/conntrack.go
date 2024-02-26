package proxy

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/store"
	"github.com/quic-go/quic-go"
)

type ConnTrack struct {
	trackedConns *SyncedMap[string, AgentID] // quic.ConnectionID => AgentID
	agentConns   *SyncedMap[AgentID, quic.Connection]
	logger       *logger.Logger

	storeClient      store.Client
	httpProxyAddress string
}

func NewConnTrack(storeClient store.Client, httpProxyAddress string) *ConnTrack {
	return &ConnTrack{
		trackedConns: NewSyncedMap[string, AgentID](),
		agentConns:   NewSyncedMap[AgentID, quic.Connection](),
		logger:       logger.GetInstance().WithFields(map[string]any{"kind": "conntrack"}),

		storeClient:      storeClient,
		httpProxyAddress: httpProxyAddress,
	}
}

func (ct *ConnTrack) OnConnStarted(connID string) {
	ct.trackedConns.Set(connID, "")
}

func (ct *ConnTrack) OnConnClose(connID string) {
	if oldAgentID, ok := ct.trackedConns.GetAndDelete(connID); ok && oldAgentID != "" {
		if oldConn, ok2 := ct.agentConns.Get(oldAgentID); ok2 && oldConn != nil {
			oldConnID := getConnID(oldConn)
			if oldConnID == connID {
				if ct.agentConns.CompareAndDelete(oldAgentID, oldConn) {
					ct.logger.Info("removed connection", slog.String("agentID", string(oldAgentID)), slog.String("connID", connID))
					if err := ct.storeClient.Delete(string(oldAgentID), ct.httpProxyAddress); err != nil {
						ct.logger.Warn("delete failure", slog.String("agentID", string(oldAgentID)), slog.String("error", err.Error()))
					}
				}
			}
		}
	}
}

func (ct *ConnTrack) PutConn(agentID AgentID, conn quic.Connection) error {
	connID := getConnID(conn)
	if connID != "" {
		if oldAgentID, ok := ct.trackedConns.Get(connID); ok && oldAgentID == "" {
			if ct.trackedConns.CompareAndSwap(connID, oldAgentID, agentID) {
				ct.logger.Info("add agent connection", slog.String("agentID", string(agentID)), slog.String("connID", connID))
			}
		}
	}
	if oldConn, ok := ct.agentConns.Swap(agentID, conn); ok && oldConn != nil {
		ct.logger.Info("closing old connection", slog.String("connID", connID))
		_ = oldConn.CloseWithError(409, "closing old connection")
	}
	// write "own" http proxy address to the store to be found by LB
	return ct.storeClient.Set(string(agentID), ct.httpProxyAddress)
}

func (ct *ConnTrack) GetConn(agentID AgentID) (previous quic.Connection, loaded bool) {
	conn, ok := ct.agentConns.Get(agentID)
	return conn, ok
}

func (ct *ConnTrack) Shutdown() {
	agentIDs, conns := ct.agentConns.Entries()
	for _, conn := range conns {
		_ = conn.CloseWithError(0, "proxy server shutdown")
	}
	for _, agentID := range agentIDs {
		err := ct.storeClient.Delete(string(agentID), ct.httpProxyAddress)
		if err != nil {
			ct.logger.Warn(fmt.Sprintf("store delete failed: %s", agentID), slog.String("error", err.Error()))
		}
	}
}

func getConnID(conn quic.Connection) string {
	if conn == nil {
		return ""
	}
	reflectValue := reflect.Indirect(reflect.ValueOf(conn))
	if reflectValue.Kind() == reflect.Struct {
		fieldVal := reflectValue.FieldByName("logID")
		if fieldVal.IsValid() {
			return fieldVal.String()
		}
	}
	return ""
}
