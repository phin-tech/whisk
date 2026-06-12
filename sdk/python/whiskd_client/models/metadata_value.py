from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MetadataValue")


@_attrs_define
class MetadataValue:
    """
    Attributes:
        type_ (str):
        bool_ (bool | Unset):
        json (str | Unset):
        number (float | Unset):
        string (str | Unset):
    """

    type_: str
    bool_: bool | Unset = UNSET
    json: str | Unset = UNSET
    number: float | Unset = UNSET
    string: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        type_ = self.type_

        bool_ = self.bool_

        json = self.json

        number = self.number

        string = self.string

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "type": type_,
            }
        )
        if bool_ is not UNSET:
            field_dict["bool"] = bool_
        if json is not UNSET:
            field_dict["json"] = json
        if number is not UNSET:
            field_dict["number"] = number
        if string is not UNSET:
            field_dict["string"] = string

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        type_ = d.pop("type")

        bool_ = d.pop("bool", UNSET)

        json = d.pop("json", UNSET)

        number = d.pop("number", UNSET)

        string = d.pop("string", UNSET)

        metadata_value = cls(
            type_=type_,
            bool_=bool_,
            json=json,
            number=number,
            string=string,
        )

        metadata_value.additional_properties = d
        return metadata_value

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
