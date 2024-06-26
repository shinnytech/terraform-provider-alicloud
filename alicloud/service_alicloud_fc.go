package alicloud

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/aliyun/fc-go-sdk"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
)

type FcService struct {
	client *connectivity.AliyunClient
}

func (s *FcService) DescribeFcService(id string) (*fc.GetServiceOutput, error) {
	response := &fc.GetServiceOutput{}
	request := &fc.GetServiceInput{ServiceName: &id}
	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetService(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "GetService", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetService", raw, requestInfo, request)
	response, _ = raw.(*fc.GetServiceOutput)
	if *response.ServiceName != id {
		err = WrapErrorf(Error(GetNotFoundMessage("FcService", id)), NotFoundMsg, FcGoSdk)
	}
	return response, err
}

func (s *FcService) WaitForFcService(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcService(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.ServiceName == id && status != Deleted {
			return nil
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.ServiceName, id, ProviderERROR)
		}
	}
}

func (s *FcService) DescribeFcFunction(id string) (*fc.GetFunctionOutput, error) {
	response := &fc.GetFunctionOutput{}
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return response, WrapError(err)
	}
	service, name := parts[0], parts[1]
	request := &fc.GetFunctionInput{
		ServiceName:  &service,
		FunctionName: &name,
	}
	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetFunction(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound", "FunctionNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "GetFunction", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetFunction", raw, requestInfo, request)
	response, _ = raw.(*fc.GetFunctionOutput)
	if *response.FunctionName == "" {
		err = WrapErrorf(Error(GetNotFoundMessage("FcFunction", id)), NotFoundMsg, FcGoSdk)
	}
	return response, err
}

func (s *FcService) WaitForFcFunction(id string, status Status, timeout int) error {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return WrapError(err)
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcFunction(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.FunctionName == parts[1] && status != Deleted {
			break
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.FunctionName, parts[1], ProviderERROR)
		}
	}
	return nil
}

func (s *FcService) DescribeFcTrigger(id string) (*fc.GetTriggerOutput, error) {
	response := &fc.GetTriggerOutput{}
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return response, WrapError(err)
	}
	service, function, name := parts[0], parts[1], parts[2]
	request := fc.NewGetTriggerInput(service, function, name)
	request.WithHeader(HeaderEnableEBTrigger, "enable")
	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetTrigger(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound", "FunctionNotFound", "TriggerNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "FcTrigger", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetTrigger", raw, requestInfo, request)
	response, _ = raw.(*fc.GetTriggerOutput)
	if *response.TriggerName != name {
		err = WrapErrorf(Error(GetNotFoundMessage("FcTrigger", name)), NotFoundMsg, ProviderERROR)
	}
	return response, err
}

// there is no DescribeServiceVersion, so emulate with ListServiceVersion
func (s *FcService) DescribeFcVersion(id string) (*fc.ListServiceVersionsOutput, error) {
	response := &fc.ListServiceVersionsOutput{}
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return response, WrapError(err)
	}
	service, version := parts[0], parts[1]
	request := &fc.ListServiceVersionsInput{
		ServiceName: &service,
		StartKey:    &version,
		Limit:       Int32Pointer(1),
	}
	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.ListServiceVersions(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound", "VersionNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "FcVersion", FcGoSdk)
		}
		return response, err
	}
	addDebug("ListServiceVersions", raw, requestInfo, request)
	response, _ = raw.(*fc.ListServiceVersionsOutput)
	if len(response.Versions) != 1 || *response.Versions[0].VersionID != version {
		err = WrapErrorf(Error(GetNotFoundMessage("FcVersion", version)), NotFoundMsg, ProviderERROR)
	}
	return response, err
}

func (s *FcService) WaitForFcVersion(id string, status Status, timeout int) error {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return WrapError(err)
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcVersion(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.Versions[0].VersionID == parts[1] {
			break
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.Versions[0].VersionID, parts[1], ProviderERROR)
		}
	}
	return nil
}

func (s *FcService) DescribeFcAlias(id string) (*fc.GetAliasOutput, error) {
	response := &fc.GetAliasOutput{}
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return response, WrapError(err)
	}
	service, name := parts[0], parts[1]
	request := fc.NewGetAliasInput(service, name)
	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetAlias(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound", "AliasNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "FcAlias", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetAlias", raw, requestInfo, request)
	response, _ = raw.(*fc.GetAliasOutput)
	if *response.AliasName != name {
		err = WrapErrorf(Error(GetNotFoundMessage("FcAlias", name)), NotFoundMsg, ProviderERROR)
	}
	return response, err
}

func (s *FcService) WaitForFcAlias(id string, status Status, timeout int) error {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return WrapError(err)
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcAlias(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.AliasName == parts[1] {
			break
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.AliasName, parts[1], ProviderERROR)
		}
	}
	return nil
}

func jsonBytesEqual(b1, b2 []byte) bool {
	var o1 interface{}
	if err := json.Unmarshal(b1, &o1); err != nil {
		return false
	}

	var o2 interface{}
	if err := json.Unmarshal(b2, &o2); err != nil {
		return false
	}

	return reflect.DeepEqual(o1, o2)
}

func (s *FcService) WaitForFcTrigger(id string, status Status, timeout int) error {
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return WrapError(err)
	}
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcTrigger(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.TriggerName == parts[2] {
			break
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.TriggerName, parts[2], ProviderERROR)
		}
	}
	return nil
}

func (s *FcService) DescribeFcCustomDomain(id string) (*fc.GetCustomDomainOutput, error) {
	request := &fc.GetCustomDomainInput{DomainName: &id}
	response := &fc.GetCustomDomainOutput{}

	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetCustomDomain(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"DomainNameNotFound"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "FcCustomDomain", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetCustomDomain", raw, requestInfo, request)
	response, _ = raw.(*fc.GetCustomDomainOutput)
	if *response.DomainName != id {
		err = WrapErrorf(Error(GetNotFoundMessage("FcCustomDomain", id)), NotFoundMsg, ProviderERROR)
	}
	return response, err
}

func (s *FcService) WaitForFcCustomDomain(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeFcCustomDomain(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if *object.DomainName == id && status != Deleted {
			return nil
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, *object.DomainName, id, ProviderERROR)
		}
	}
}

func (s *FcService) DescribeFcFunctionAsyncInvokeConfig(id string) (*fc.GetFunctionAsyncInvokeConfigOutput, error) {
	serviceName, functionName, qualifier, err := parseFCDestinationConfigId(id)
	if err != nil {
		return nil, err
	}
	request := &fc.GetFunctionAsyncInvokeConfigInput{
		ServiceName:  &serviceName,
		FunctionName: &functionName,
	}
	if qualifier != "" {
		request.Qualifier = &qualifier
	}
	response := &fc.GetFunctionAsyncInvokeConfigOutput{}

	var requestInfo *fc.Client
	raw, err := s.client.WithFcClient(func(fcClient *fc.Client) (interface{}, error) {
		requestInfo = fcClient
		return fcClient.GetFunctionAsyncInvokeConfig(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"ServiceNotFound", "FunctionNotFound", "AsyncConfigNotExists"}) {
			err = WrapErrorf(err, NotFoundMsg, FcGoSdk)
		} else {
			err = WrapErrorf(err, DefaultErrorMsg, id, "FcFunctionAsyncInvokeConfig", FcGoSdk)
		}
		return response, err
	}
	addDebug("GetFunctionAsyncInvokeConfig", raw, requestInfo, request)
	response, _ = raw.(*fc.GetFunctionAsyncInvokeConfigOutput)
	return response, err
}

func (s *FcService) WaitForFcFunctionAsyncInvokeConfig(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		_, err := s.DescribeFcFunctionAsyncInvokeConfig(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if status != Deleted {
			return nil
		}
		time.Sleep(DefaultIntervalShort * time.Second)
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, id, ProviderERROR)
		}
	}
}
