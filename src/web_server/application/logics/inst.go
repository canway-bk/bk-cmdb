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

package logics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/rentiansheng/xlsx"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	lang "configcenter/src/common/language"
	webCommon "configcenter/src/web_server/common"
	"reflect"
)

//GetImportInsts get insts from excel file
func GetImportInsts(f *xlsx.File, objID, url string, header http.Header, headerRow int, isInst bool, defLang lang.DefaultCCLanguageIf) (map[int]map[string]interface{}, []string, error) {

	fields, err := GetObjFieldIDs(objID, url, nil, header)
	if nil != err {
		return nil, nil, errors.New(defLang.Languagef("web_get_object_field_failure", err.Error()))
	}
	if 0 == len(f.Sheets) {
		blog.Error("the excel file sheets is empty")
		return nil, nil, errors.New(defLang.Language("web_excel_content_empty"))
	}
	sheet := f.Sheets[0]
	if nil == sheet {
		blog.Error("the excel fiel sheet is nil")
		return nil, nil, errors.New(defLang.Language("web_excel_sheet_not_found"))
	}
	if isInst {
		return GetExcelData(sheet, fields, common.KvMap{"import_from": common.HostAddMethodExcel}, true, headerRow, defLang)
	} else {
		return GetRawExcelData(sheet, common.KvMap{"import_from": common.HostAddMethodExcel}, headerRow, defLang)
	}
}

//GetInstData get inst data
func GetInstData(ownerID, objID, instIDStr, apiAddr string, header http.Header, kvMap map[string]string) ([]interface{}, error) {

	instInfo := make([]interface{}, 0)
	sInstCond := make(map[string]interface{})
	instIDArr := strings.Split(instIDStr, ",")

	iInstIDArr := make([]int, 0)
	for _, j := range instIDArr {
		instID, _ := strconv.Atoi(j)
		iInstIDArr = append(iInstIDArr, instID)
	}

	// construct the search condition

	sInstCond["fields"] = []string{}
	sInstCond["condition"] = map[string]interface{}{
		common.BKInstIDField: map[string]interface{}{
			"$in": iInstIDArr,
		},
		common.BKOwnerIDField: ownerID,
		common.BKObjIDField:   objID,
	}
	sInstCond["page"] = nil

	// read insts
	url := apiAddr + fmt.Sprintf("/api/%s/inst/search/owner/%s/object/%s/detail", webCommon.API_VERSION, ownerID, objID)
	result, _ := httpRequest(url, sInstCond, header)
	blog.Info("search inst  url:%s", url)
	blog.Info("search inst  return:%s", result)
	js, _ := simplejson.NewJson([]byte(result))
	instData, _ := js.Map()
	instResult := instData["result"].(bool)
	if !instResult {
		return nil, errors.New(instData["bk_error_msg"].(string))
	}

	instDataArr := instData["data"].(map[string]interface{})
	instInfo = instDataArr["info"].([]interface{})
	instCnt, _ := instDataArr["count"].(json.Number).Int64()
	if !instResult || 0 == instCnt {
		return instInfo, errors.New("no inst")
	}

	// read object attributes
	url = apiAddr + fmt.Sprintf("/api/%s/object/attr/search", webCommon.API_VERSION)
	attrCond := make(map[string]interface{})
	attrCond[common.BKObjIDField] = objID
	attrCond[common.BKOwnerIDField] = ownerID
	result, _ = httpRequest(url, attrCond, header)
	blog.Info("get inst attr  url:%s", url)
	blog.Info("get inst attr return:%s", result)
	js, _ = simplejson.NewJson([]byte(result))
	instAttr, _ := js.Map()
	attrData := instAttr["data"].([]interface{})
	for _, j := range attrData {
		cell := j.(map[string]interface{})
		key := cell[common.BKPropertyIDField].(string)
		value, ok := cell[common.BKPropertyNameField].(string)
		if ok {
			kvMap[key] = value
		} else {
			kvMap[key] = ""
		}

	}
	return instInfo, nil
}
// 获取转换为markdown后的实例数据
func GetInstDataConvertMarkDown(instInfo []interface{}, propertyMap map[string]Property)([]interface{}, error) {
	blog.Debug("GetInstDataConvertList start")
	instInfoM := make([]interface{}, 0)
	for _, info := range instInfo {
		infoMap := info.(map[string]interface{})
		for k, v := range infoMap {
			switch v.(type) {
			case []interface{}:
				if propertyMap[k].PropertyType != common.FieldTypeList {
					break
				}
				option := propertyMap[k].Option
				headers, ok:= option.([]interface{})
				if !ok {
					blog.Error("option is not lst")
					return nil, nil
				}
				value, err := ListConvertMarkDown(v.([]interface{}), headers)
				if nil != err {
					blog.Error("ListConvertMarkDown error:%v", err.Error())
					return nil, err
				}
				infoMap[k] = value
			}
		}
		instInfoM = append(instInfoM, infoMap)
	}
	return instInfoM ,nil
}

// 列表转换为markdown
func ListConvertMarkDown(values []interface{},headers []interface{}) (string, error) {
	blog.Debug("ListConvertMarkDown start")
    headerColumnLenMap := GetColumnLen(headers, values)
    headerStr := "| "
    pointStr := "| "
    for _,h := range headers {
		hMap := h.(map[string]interface{})
        headerName := hMap["list_header_name"].(string)
        headerSpace := GetStrSpace(headerName, headerColumnLenMap[headerName], "header")
        headerStr = headerStr + headerName + headerSpace + " | "
        drop := GetPointDrop(len(headerName), "left")
        pointSpace := GetStrSpace(headerName, headerColumnLenMap[headerName], "point")
        pointStr = pointStr + drop + pointSpace +" | "
    }
    valueStr := ""
    for _, value := range  values {
        valueMap := value.(map[string]interface{})
        vStr := "| "
        for _,h := range headers {
			hMap := h.(map[string]interface{})
            headerName := hMap["list_header_name"].(string)
            for k, v := range valueMap {
                if k == headerName {
                    valueSpace := GetStrSpace(fmt.Sprintf("%v",v), headerColumnLenMap[headerName],"value")
                    vStr = vStr + fmt.Sprintf("%v",v) + valueSpace + " | "
                }
            }
        }
        if "" != valueStr {
            valueStr = valueStr + "\n" + vStr
        }else{
            valueStr = valueStr + vStr
        }
    }
    markDownStr := headerStr + "\n" + pointStr + "\n" + valueStr
   	blog.Debug("\n markDownStr:%v",markDownStr)
	return markDownStr, nil
}

// 获取列表的列名称长度(以一列中字符长度最长的为准)
func GetColumnLen(headers []interface{}, values[]interface{}) (map[string]int) {
    columnLenMap := make(map[string]int)
    for _,h := range headers {
		hMap := h.(map[string]interface{})
        headerName := hMap["list_header_name"].(string)
		headerLen := len(headerName)
        for _,v := range values {
            vMap := v.(map[string]interface{})
            headerNameLen := len(vMap[headerName].(string))
            if headerLen < headerNameLen {
                headerLen = headerNameLen
            }
        }
        columnLenMap[headerName] = headerLen
    }
    return columnLenMap
}

func GetStrSpace(name string, headerLen int, title string) (string) {
    if (headerLen < len(name)) {
        return ""
    }
    spaceLen := headerLen-len(name)
    spaceStr := ""
    for i:=0;i<spaceLen;i++ {
        if title == "point"{
            spaceStr = spaceStr + "-"
        }else {
            spaceStr = spaceStr + " "
        }
    }
    return spaceStr
}

func GetPointDrop(num int, point string) string {
    drop := ""
    if point == "center" {
        drop = ":"
    }
    for a:=0; a<num; a++ {
        drop = drop + "-"
    }
    if point == "right" || point == "center"{
            drop = drop + ":"
        }
    return drop
}

// 获取markdown转换为列表的实例数据
func GetMarkConvertList(instInfo map[int]map[string]interface{}, propertyMap map[string]Property) (int, string) {
	for _,info := range instInfo {
		for k, v := range info {
			if _,ok := propertyMap[k];ok && propertyMap[k].PropertyType == "list" {
				values, importHeaderMap := markConvertList(v.(string))
				if 0 != len(importHeaderMap) {
					errHeaderName := checkHeaders(importHeaderMap, propertyMap[k].Option)
					blog.Debug("errHeaderName: %v",errHeaderName)
					if "" != errHeaderName{
						return common.CCErrWebMarkDownConvertListFail, errHeaderName
					}
				}
				info[k] = values
			}
		}
	}
	blog.Debug("instInfos:%v",instInfo)
	return 0, ""
}

// markDown转换为列表
func markConvertList(str string) ([]map[string]interface{}, map[int]string){
	strList := strings.Split(str, "\n")
	headerStrList := strings.Split(strings.Trim(strList[0], "|"), "|")
	headerMap := make(map[int]string)
	for index, headerStr := range headerStrList{
		headerMap[index] = strings.Trim(headerStr, " ")
	}
	values := make([]map[string]interface{}, 0)
	for i:=2;i<len(strList);i++ {
		valueStrList := strings.Split(strings.Trim(strList[i], "|"), "|")
		valueMap := make(map[string]interface{})
		for index, valueStr := range valueStrList {
			if _,ok := headerMap[index];ok{
				if ""==headerMap[index]{
					continue
				}
				valueMap[headerMap[index]] = strings.Trim(valueStr," ")
			}
		}
		values = append(values, valueMap)
	}
	blog.Debug("values:%v", values)
	return values, headerMap
}

// 校验表头是否正确
func checkHeaders(importHeaderMap map[int]string, option interface{}) string {
	blog.Debug("checkHeaders start")
	op ,ok := option.([]interface{})
	if !ok {
		blog.Error("checkHeaders err:option is not []interface{},type:", reflect.TypeOf(option))
		return ""
	}
	headerMap := make(map[string]interface{})
	for _,h := range op {
		hMap := h.(map[string]interface{})
		headerMap[hMap["list_header_name"].(string)] = headerMap[hMap["list_header_describe"].(string)]
	}
	errHeaderName := ""
	for _, headerName := range importHeaderMap {
		blog.Debug("headerName:%v",headerName)
		if _,ok := headerMap[headerName]; !ok {
			errHeaderName = errHeaderName + headerName
			if "" != errHeaderName {
				errHeaderName = errHeaderName + ";"
			}
		}
	}
	return errHeaderName
}


// 导入时去除密码类型字段
func ImportMovePassword(instInfo map[int]map[string]interface{}, propertyMap map[string]Property) () {
	for _,info := range instInfo {
		for k, _ := range info {
			if _,ok := propertyMap[k];ok && propertyMap[k].PropertyType == "password" {
				delete(info,k)
			}
		}
	}
}

// 导出时去除密码类型字段
func ExportMovePassword(instInfo []interface{}, propertyMap map[string]Property) () {
	for _, info := range instInfo {
		infoMap := info.(map[string]interface{})
		for k, _ := range infoMap {
			if _,ok := propertyMap[k];ok && propertyMap[k].PropertyType == "password" {
				delete(infoMap,k)
			}
		}
	}
}