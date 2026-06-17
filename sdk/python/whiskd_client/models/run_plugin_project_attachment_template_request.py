from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.run_plugin_project_attachment_template_request_values import (
        RunPluginProjectAttachmentTemplateRequestValues,
    )


T = TypeVar("T", bound="RunPluginProjectAttachmentTemplateRequest")


@_attrs_define
class RunPluginProjectAttachmentTemplateRequest:
    """
    Attributes:
        project_id (str):
        values (RunPluginProjectAttachmentTemplateRequestValues | Unset):
    """

    project_id: str
    values: RunPluginProjectAttachmentTemplateRequestValues | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        project_id = self.project_id

        values: dict[str, Any] | Unset = UNSET
        if not isinstance(self.values, Unset):
            values = self.values.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "projectId": project_id,
            }
        )
        if values is not UNSET:
            field_dict["values"] = values

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.run_plugin_project_attachment_template_request_values import (
            RunPluginProjectAttachmentTemplateRequestValues,
        )

        d = dict(src_dict)
        project_id = d.pop("projectId")

        _values = d.pop("values", UNSET)
        values: RunPluginProjectAttachmentTemplateRequestValues | Unset
        if isinstance(_values, Unset):
            values = UNSET
        else:
            values = RunPluginProjectAttachmentTemplateRequestValues.from_dict(_values)

        run_plugin_project_attachment_template_request = cls(
            project_id=project_id,
            values=values,
        )

        run_plugin_project_attachment_template_request.additional_properties = d
        return run_plugin_project_attachment_template_request

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
