package httpapi

import (
	"log/slog"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// AttachObjectStore 挂载 SQLite 对象库：启动时加载，写路径同步落盘。
// store 为 nil 时保持纯内存（测试默认）。
func (s *Server) AttachObjectStore(store *storage.ObjectStore, log *slog.Logger) {
	s.objects = store
	if store == nil {
		return
	}
	if log == nil {
		log = s.log
	}
	if err := s.loadPersistedObjects(log); err != nil {
		log.Warn("objectstore load failed", "error", err)
	}
}

func (s *Server) persistResource(kind resource.Kind, id string, payload map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("resource:"+string(kind), id, payload)
}

func (s *Server) persistResourceDelete(kind resource.Kind, id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("resource:"+string(kind), id)
	_ = s.objects.Delete("resource-info:"+string(kind), id)
	_ = s.objects.Delete("resource-data:"+string(kind), id)
}

func (s *Server) persistInfo(kind resource.Kind, id string, payload map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("resource-info:"+string(kind), id, payload)
}

func (s *Server) persistInfoDelete(kind resource.Kind, id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("resource-info:"+string(kind), id)
}

func (s *Server) persistData(kind resource.Kind, id string, payload map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("resource-data:"+string(kind), id, payload)
}

func (s *Server) persistDataDelete(kind resource.Kind, id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("resource-data:"+string(kind), id)
}

func (s *Server) persistCatalogUpsert(bucket, idField string, objs []map[string]any) {
	if s.objects == nil {
		return
	}
	for _, o := range objs {
		id, _ := o[idField].(string)
		if id == "" {
			continue
		}
		_ = s.objects.Put(bucket, id, o)
	}
}

func (s *Server) persistCatalogDelete(bucket, id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete(bucket, id)
}

func (s *Server) persistVIIDServer(id string, obj map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("viidserver", id, obj)
}

func (s *Server) persistVIIDServerDelete(id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("viidserver", id)
}

func (s *Server) persistSubscribe(id string, obj map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("subscribe", id, obj)
}

func (s *Server) persistSubscribeDelete(id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("subscribe", id)
}

func (s *Server) persistDisposition(id string, obj map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("disposition", id, obj)
}

func (s *Server) persistDispositionDelete(id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("disposition", id)
}

func (s *Server) persistVIASTask(id string, obj map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("vias:task", id, obj)
}

func (s *Server) persistVIASTaskDelete(id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("vias:task", id)
}

func (s *Server) persistVIASRule(id string, obj map[string]any) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Put("vias:rule", id, obj)
}

func (s *Server) persistVIASRuleDelete(id string) {
	if s.objects == nil {
		return
	}
	_ = s.objects.Delete("vias:rule", id)
}

func (s *Server) loadPersistedObjects(log *slog.Logger) error {
	// Resources
	for _, kind := range resource.AllKinds {
		items, err := s.objects.List("resource:" + string(kind))
		if err != nil {
			return err
		}
		idField := resource.IDOf(kind)
		for _, m := range items {
			id, _ := m[idField].(string)
			if id == "" {
				continue
			}
			s.repo.data[kind][id] = m
		}
		infos, err := s.objects.List("resource-info:" + string(kind))
		if err != nil {
			return err
		}
		for _, m := range infos {
			id, _ := m[idField].(string)
			if id == "" {
				continue
			}
			s.repo.info[kind][id] = m
		}
		blobIDs, err := s.objects.ListIDs("resource-data:" + string(kind))
		if err != nil {
			return err
		}
		for _, id := range blobIDs {
			m, ok, err := s.objects.Get("resource-data:"+string(kind), id)
			if err != nil || !ok {
				continue
			}
			s.repo.blob[kind][id] = m
		}
	}

	loadCat := func(bucket string, dest map[string]map[string]any, idField string) error {
		items, err := s.objects.List(bucket)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		// 有持久化数据时清空种子，避免与磁盘冲突。
		for k := range dest {
			delete(dest, k)
		}
		for _, m := range items {
			id, _ := m[idField].(string)
			if id == "" {
				continue
			}
			dest[id] = m
		}
		return nil
	}
	if err := loadCat("catalog:APE", s.catalog.apes, "APEID"); err != nil {
		return err
	}
	if err := loadCat("catalog:APS", s.catalog.apss, "APSID"); err != nil {
		return err
	}
	if err := loadCat("catalog:Tollgate", s.catalog.tollgates, "TollgateID"); err != nil {
		return err
	}
	if err := loadCat("catalog:Lane", s.catalog.lanes, "LaneID"); err != nil {
		return err
	}

	if servers, err := s.objects.List("viidserver"); err != nil {
		return err
	} else if len(servers) > 0 {
		s.servers.mu.Lock()
		s.servers.data = map[string]map[string]any{}
		for _, m := range servers {
			id, _ := m["VIIDServerID"].(string)
			if id != "" {
				s.servers.data[id] = m
			}
		}
		s.servers.mu.Unlock()
	}

	if subs, err := s.objects.List("subscribe"); err != nil {
		return err
	} else {
		s.subRepo.mu.Lock()
		for _, m := range subs {
			id, _ := m["SubscribeID"].(string)
			if id != "" {
				s.subRepo.subs[id] = m
			}
		}
		s.subRepo.mu.Unlock()
	}

	if disps, err := s.objects.List("disposition"); err != nil {
		return err
	} else {
		s.subRepo.mu.Lock()
		for _, m := range disps {
			id, _ := m["DispositionID"].(string)
			if id != "" {
				s.subRepo.disp[id] = m
			}
		}
		s.subRepo.mu.Unlock()
	}

	if tasks, err := s.objects.List("vias:task"); err != nil {
		return err
	} else {
		s.vias.mu.Lock()
		for _, m := range tasks {
			id, _ := m["TaskID"].(string)
			if id != "" {
				s.vias.tasks[id] = m
			}
		}
		s.vias.mu.Unlock()
	}

	if rules, err := s.objects.List("vias:rule"); err != nil {
		return err
	} else {
		s.vias.mu.Lock()
		for _, m := range rules {
			id, _ := m["AnalysisRuleID"].(string)
			if id != "" {
				s.vias.rules[id] = m
			}
		}
		s.vias.mu.Unlock()
	}

	log.Info("objectstore loaded")
	return nil
}
