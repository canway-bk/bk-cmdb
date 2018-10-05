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

package object

import (
	"configcenter/src/common"
	"configcenter/src/common/bkbase"
	"configcenter/src/common/blog"
	"configcenter/src/common/core/cc/actions"
	"configcenter/src/common/util"
	"configcenter/src/scene_server/topo_server/topo_service/manager"
	api "configcenter/src/source_controller/api/object"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"configcenter/src/common/errors"
	"github.com/emicklei/go-restful"
)

var objatt = &objattAction{}

// HostAction
type objattAction struct {
	base.BaseAction
	mgr manager.Manager
}

func init() {

	// register action

	actions.RegisterNewAction(actions.Action{Verb: common.HTTPCreate, Path: "/objectattr", Params: nil, Handler: objatt.CreateObjectAtt})
	actions.RegisterNewAction(actions.Action{Verb: common.HTTPSelectPost, Path: "/objectattr/search", Params: nil, Handler: objatt.SelectObjectAttWithParams})
	actions.RegisterNewAction(actions.Action{Verb: common.HTTPUpdate, Path: "/objectattr/{id}", Params: nil, Handler: objatt.UpdateObjectAtt})
	actions.RegisterNewAction(actions.Action{Verb: common.HTTPDelete, Path: "/objectattr/{id}", Params: nil, Handler: objatt.DeleteObjectAtt})

	// set object att
	objatt.CreateAction()
	// set httpclient
	manager.SetManager(objatt)
}

// SetManager implement the manager's Hooker interface
func (cli *objattAction) SetManager(mgr manager.Manager) error {
	cli.mgr = mgr
	return nil
}

func (cli *objattAction) CreateObjectAtt(req *restful.Request, resp *restful.Response) {

	blog.Info("create objectatt information")
	// get the language
	language := util.GetActionLanguage(req)
	// get the error info by the language
	defErr := cli.CC.Error.CreateDefaultCCErrorIf(language)
	forward := &api.ForwardParam{Header: req.Request.Header}

	// execute
	cli.CallResponseEx(func() (int, interface{}, error) {

		// read data
		val, err := ioutil.ReadAll(req.Request.Body)
		if err != nil {
			blog.Error("failed to read request body, error info is %s", err.Error())
			return http.StatusBadRequest, nil, defErr.Error(common.CCErrCommHTTPReadBodyFailed)
		}

		var obj api.ObjAttDes
		if jsErr := json.Unmarshal(val, &obj); nil != jsErr {
			blog.Error("unmarshal json failed, error information is %v", jsErr)
			return http.StatusBadRequest, nil, defErr.Error(common.CC_ERR_Comm_JSON_DECODE)
		}

		blog.Debug("create %s", string(val))

		// check list header name begin(chace)
		if obj.PropertyType == common.FieldTypeList {
			headerList, ok :=  obj.Option.([]interface{})
			if !ok {
				return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeUpdateFailed)
			}
			errorData := CheckListHeaderChart(headerList, defErr)
			if 0 != len(errorData) {
				return http.StatusInternalServerError, errorData, defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameCheck)
			}
			errorData = CheckListHeaderRepeat(headerList, defErr)
			if 0 != len(errorData) {
				return http.StatusInternalServerError, errorData, defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameCheck)
			}
		}
		// check list header name end(chace)

		// deal data
		result, ctrErr := cli.mgr.CreateObjectAtt(forward, obj, defErr)
		if nil == ctrErr {
			return http.StatusOK, map[string]int{"id": result}, nil
		}
		blog.Error("create objectatt failed, error information is %s", ctrErr.Error())
		return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeCreateFailed)
	}, resp)
}

func (cli *objattAction) SelectObjectAttWithParams(req *restful.Request, resp *restful.Response) {

	blog.Info("select objectatt information with params")
	// get the language
	language := util.GetActionLanguage(req)
	// get the error info by the language
	defErr := cli.CC.Error.CreateDefaultCCErrorIf(language)
	forward := &api.ForwardParam{Header: req.Request.Header}

	// execute
	cli.CallResponseEx(func() (int, interface{}, error) {

		val, err := ioutil.ReadAll(req.Request.Body)
		if err != nil {
			blog.Error("failed to read request body, error info is %s", err.Error())
			return http.StatusBadRequest, nil, defErr.Error(common.CCErrCommHTTPReadBodyFailed)
		}

		// deal request
		result, ctrErr := cli.mgr.SelectObjectAtt(forward, val, defErr)
		if nil == ctrErr {
			return http.StatusOK, result, nil
		}

		blog.Error("search objectatt failed, error information is %v", ctrErr)
		return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeSelectFailed)

	}, resp)
}

func (cli *objattAction) UpdateObjectAtt(req *restful.Request, resp *restful.Response) {

	blog.Info("update objectatt information")

	// get the language
	language := util.GetActionLanguage(req)
	// get the error info by the language
	defErr := cli.CC.Error.CreateDefaultCCErrorIf(language)
	forward := &api.ForwardParam{Header: req.Request.Header}

	// execute
	cli.CallResponseEx(func() (int, interface{}, error) {

		val, err := ioutil.ReadAll(req.Request.Body)
		if err != nil {
			blog.Error("failed to read request body, error info is %s", err.Error())
			return http.StatusBadRequest, nil, defErr.Error(common.CCErrCommHTTPReadBodyFailed)
		}
		// check list header name start (chace)
		obj := make(map[string]interface{})
		json.Unmarshal(val, &obj)
		if obj[common.BKPropertyTypeField] == common.FieldTypeList {
			headerList, ok := obj["option"].([]interface{})
			if !ok {
				return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeUpdateFailed)
			}
			errorData := CheckListHeaderChart(headerList, defErr)
			if 0 != len(errorData) {
				return http.StatusInternalServerError, errorData, defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameCheck)
			}
			errorData = CheckListHeaderRepeat(headerList, defErr)
			if 0 != len(errorData) {
				return http.StatusInternalServerError, errorData, defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameCheck)
			}
		}
		// check list header name end

		// get params
		attrID, attrIDErr := strconv.Atoi(req.PathParameter("id"))
		if nil != attrIDErr {
			blog.Error("attrid(%s) is invalid, error info is %s", req.PathParameter("id"), attrIDErr.Error())
			return http.StatusBadRequest, nil, defErr.Errorf(common.CCErrCommParamsNeedInt, "id")
		}

		// deal data
		ctrErr := cli.mgr.UpdateObjectAtt(forward, attrID, val, defErr)
		if nil == ctrErr {
			return http.StatusOK, nil, nil
		}

		blog.Error("update objectattr failed, error: %v", ctrErr)
		return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeUpdateFailed)

	}, resp)

}

func (cli *objattAction) DeleteObjectAtt(req *restful.Request, resp *restful.Response) {

	blog.Info("delete objectatt information")

	// get the language
	language := util.GetActionLanguage(req)
	// get the error info by the language
	defErr := cli.CC.Error.CreateDefaultCCErrorIf(language)
	forward := &api.ForwardParam{Header: req.Request.Header}

	// execute
	cli.CallResponseEx(func() (int, interface{}, error) {

		val, err := ioutil.ReadAll(req.Request.Body)
		if err != nil {
			blog.Error("failed to read request body, error info is %s", err.Error())
			return http.StatusBadRequest, nil, defErr.Error(common.CCErrCommHTTPReadBodyFailed)
		}

		// get params
		attrID, attrIDErr := strconv.Atoi(req.PathParameter("id"))
		if nil != attrIDErr {
			blog.Error("id(%s) is invalid, error info is %s", req.PathParameter("id"), attrIDErr.Error())
			return http.StatusBadRequest, nil, defErr.Errorf(common.CCErrCommParamsNeedInt, "id")
		}

		// deal data
		ctrErr := cli.mgr.DeleteObjectAtt(forward, attrID, val, defErr)
		if nil == ctrErr {
			return http.StatusOK, nil, nil
		}

		blog.Error("update objectattr failed, error: %v", ctrErr)
		return http.StatusInternalServerError, nil, defErr.Error(common.CCErrTopoObjectAttributeDeleteFailed)
	}, resp)

}



// helper
// 校验模型实例字段list类型的字段名称是否为字符串
func CheckListHeaderChart (headerList []interface{}, defErr errors.DefaultCCErrorIf) map[int]string {
	errorData := make(map[int]string)
	for index, header := range headerList {
		headerMap := header.(map[string]interface{})
		headerName, ok:= headerMap["list_header_name"].(string)
		if !ok {
			errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameField).Error()
			continue
		}
		if "" == headerName {
			errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameEmpty).Error()
			continue
		}
		//if !util.IsField(headerName) {
		//	errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameField).Error()
		//	continue
		//}
		if !util.CheckLen(headerName, 2, 40) {
			errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameLen).Error()
			continue
		}
	}
	return errorData
}

// 校验模型实例字段list类型的字段名称是否重复
func CheckListHeaderRepeat(headerList []interface{}, defErr errors.DefaultCCErrorIf) map[int]string {
	errorData := make(map[int]string)
	checkData := make(map[string]int, 0)
	for index, header := range headerList {
		headerMap := header.(map[string]interface{})
		headerName, ok:= headerMap["list_header_name"].(string)
		if !ok {
			errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameField).Error()
			continue
		}
		if _, ok := checkData[headerName]; !ok {
			checkData[headerName] = index
		}else {
			for index, header := range headerList {
				headerMap := header.(map[string]interface{})
				if headerMap["list_header_name"] == headerName {
					if _,ok := errorData[index]; !ok {
						errorData[index] = defErr.Error(common.CCErrTopoObjectAttrCreateTypeListNameRepeat).Error()
					}
				}
			}
		}
	}
	return errorData
}