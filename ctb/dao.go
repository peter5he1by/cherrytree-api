package ctb

import (
	"sort"
)

func (r Handle) selectNodeMetaById(id int32) (ptNodeMeta, error) {
	var nm ptNodeMeta
	result := r.db.Model(&tNode{}).Where("node_id = ?", id).Take(&nm)
	return nm, result.Error
}

func (r Handle) selectNodeMetaByIds(ids ...int32) ([]ptNodeMeta, error) {
	var list []ptNodeMeta
	for _, id := range ids {
		nm, err := r.selectNodeMetaById(id)
		if err != nil {
			return nil, err
		}
		list = append(list, nm)
	}
	return list, nil
}

func (r Handle) selectChildrenByNodeId(nodeId int32) (tChildren, error) {
	var tc tChildren
	result := r.db.Where("node_id = ?", nodeId).Take(&tc)
	return tc, result.Error
}

func (r Handle) selectChildrenByFatherId(nodeId int32) ([]tChildren, error) {
	var list []tChildren
	result := r.db.Where("father_id = ?", nodeId).Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Sequence < list[j].Sequence
	})
	return list, nil
}

func (r Handle) selectNodeContentById(id int32) (*ptNodeContent, error) {
	var raw ptNodeContent
	result := r.db.Model(&tNode{}).Where("node_id = ?", id).Take(&raw)
	return &raw, result.Error
}

func (r Handle) selectImagesByNodeId(id int32) ([]*tImage, error) {
	var images []*tImage
	result := r.db.Where("node_id = ?", id).Find(&images)
	if result.Error != nil {
		return nil, result.Error
	}
	return images, nil
}
