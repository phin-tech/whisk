from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="PluginTemplateField")


@_attrs_define
class PluginTemplateField:
    """
    Attributes:
        id (str):
        label (str):
        type_ (str):
        options (list[str] | Unset):
        placeholder (str | Unset):
        required (bool | Unset):
    """

    id: str
    label: str
    type_: str
    options: list[str] | Unset = UNSET
    placeholder: str | Unset = UNSET
    required: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        type_ = self.type_

        options: list[str] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options

        placeholder = self.placeholder

        required = self.required

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "label": label,
                "type": type_,
            }
        )
        if options is not UNSET:
            field_dict["options"] = options
        if placeholder is not UNSET:
            field_dict["placeholder"] = placeholder
        if required is not UNSET:
            field_dict["required"] = required

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        type_ = d.pop("type")

        options = cast(list[str], d.pop("options", UNSET))

        placeholder = d.pop("placeholder", UNSET)

        required = d.pop("required", UNSET)

        plugin_template_field = cls(
            id=id,
            label=label,
            type_=type_,
            options=options,
            placeholder=placeholder,
            required=required,
        )

        plugin_template_field.additional_properties = d
        return plugin_template_field

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
