/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"configcenter/src/common/condition"
	"strconv"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	frtypes "configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	gparams "configcenter/src/common/paraparse"
	"configcenter/src/scene_server/topo_server/core/operation"
	"configcenter/src/scene_server/topo_server/core/types"
)

// CreateSet create a new set
func (s *topoService) CreateSet(params types.ContextParams, pathParams, queryParams ParamsGetter, data frtypes.MapStr) (interface{}, error) {

	obj, err := s.core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, err
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	return s.core.SetOperation().CreateSet(params, obj, bizID, data)
}
func (s *topoService) DeleteSets(params types.ContextParams, pathParams, queryParams ParamsGetter, data frtypes.MapStr) (interface{}, error) {
	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	obj, err := s.core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDSet)

	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, err
	}

	cond := &operation.OpCondition{}
	if err = data.MarshalJSONInto(cond); nil != err {
		blog.Errorf("[api-set] failed to parse to the operation condition, error info is %s", err.Error())
		return nil, params.Err.New(common.CCErrCommParamsIsInvalid, err.Error())
	}

	return nil, s.core.SetOperation().DeleteSet(params, obj, bizID, cond.Delete.InstID)
}

// DeleteSet delete the set
func (s *topoService) DeleteSet(params types.ContextParams, pathParams, queryParams ParamsGetter, data frtypes.MapStr) (interface{}, error) {

	if "batch" == pathParams("set_id") {
		return s.DeleteSets(params, pathParams, queryParams, data)
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	obj, err := s.core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDSet)

	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, err
	}

	return nil, s.core.SetOperation().DeleteSet(params, obj, bizID, []int64{setID})
}

// UpdateSet update the set
func (s *topoService) UpdateSet(params types.ContextParams, pathParams, queryParams ParamsGetter, data frtypes.MapStr) (interface{}, error) {

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	obj, err := s.core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, err
	}

	hostCond := &metadata.QueryInput{
		Condition: condition.CreateCondition().Field(common.BKSetIDField).Eq(setID).ToMapStr(),
	}
	count, sets, err := s.core.SetOperation().FindSet(params, obj, hostCond)
	if err != nil {
		blog.Errorf("[api-set]failed, get set by id failed, err: %+v", err)
		return nil, err
	}
	if count == 0 {
		blog.Errorf("[api-set]failed, get set by id failed, got empty, setID:%d", setID)
		return nil, params.Err.Errorf(common.CCErrCommParamsIsInvalid, "set id")
	}
	if len(sets) > 1 {
		blog.Errorf("[api-set]failed, get set by id failed, got %d, setID:%d", count, setID)
		return nil, params.Err.Errorf(common.CCErrCommParamsIsInvalid, "set id")
	}
	set := sets[0]
	
	// check name unique constraint
	if _, exist := data[common.BKSetNameField]; exist == true {
		searchCondition := condition.CreateCondition().Field(common.BKAppIDField).Eq(bizID)
		parentID, err := set.GetParentID()
		if err != nil {
			blog.Errorf("[api-set]failed, get set by id failed, got %d, setID:%d", count, setID)
			return nil, params.Err.Errorf(common.CCErrCommInstFieldConvFail, "set", common.BKInstParentStr, "int", err.Error())
		}
		searchCondition.Field(common.BKInstParentStr).Eq(parentID)
		searchCondition.Field(common.BKSetNameField).Eq(data[common.BKSetNameField])
		searchCondition.Field(common.BKSetIDField).NotEq(setID)
		cond := &metadata.QueryInput{
			Condition: searchCondition.ToMapStr(),
		}
		count, _, err := s.core.SetOperation().FindSet(params, obj, cond)
		if err != nil {
			blog.Errorf("[api-set]failed, filter sets by condition failed, err: %+v", err)
			return nil, err
		}
		if count > 0 {
			blog.Errorf("[api-set]failed, rename set failed, set name repeated, set: %+v", set)
			return nil, params.Err.Errorf(common.CCErrTopoSetNameRepeated)
		}
	}

	return nil, s.core.SetOperation().UpdateSet(params, data, obj, bizID, setID)
}

// SearchSet search the set
func (s *topoService) SearchSet(params types.ContextParams, pathParams, queryParams ParamsGetter, data frtypes.MapStr) (interface{}, error) {

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-set]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	obj, err := s.core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDSet)
	if nil != err {
		blog.Errorf("[api-set]failed to search the set, %s", err.Error())
		return nil, err
	}

	paramsCond := &gparams.SearchParams{
		Condition: mapstr.New(),
	}
	if err = data.MarshalJSONInto(paramsCond); nil != err {
		return nil, err
	}

	paramsCond.Condition[common.BKAppIDField] = bizID
	paramsCond.Condition[common.BKOwnerIDField] = params.SupplierAccount

	queryCond := &metadata.QueryInput{}
	queryCond.Condition = paramsCond.Condition
	queryCond.Fields = strings.Join(paramsCond.Fields, ",")
	page := metadata.ParsePage(paramsCond.Page)
	queryCond.Start = page.Start
	queryCond.Sort = page.Sort
	queryCond.Limit = page.Limit

	cnt, instItems, err := s.core.SetOperation().FindSet(params, obj, queryCond)
	if nil != err {
		blog.Errorf("[api-set] failed to find the objects(%s), error info is %s", pathParams("obj_id"), err.Error())
		return nil, err
	}

	result := frtypes.MapStr{}
	result.Set("count", cnt)
	result.Set("info", instItems)

	return result, nil

}
