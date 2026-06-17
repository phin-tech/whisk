from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.plugin_resolver import PluginResolver
    from ..models.project_attachment_template import ProjectAttachmentTemplate


T = TypeVar("T", bound="PluginStatus")


@_attrs_define
class PluginStatus:
    """
    Attributes:
        dir_ (str):
        id (str):
        manifest_path (str):
        name (str):
        trusted (bool):
        valid (bool):
        version (str):
        error (str | Unset):
        project_attachment_templates (list[ProjectAttachmentTemplate] | Unset):
        resolvers (list[PluginResolver] | Unset):
    """

    dir_: str
    id: str
    manifest_path: str
    name: str
    trusted: bool
    valid: bool
    version: str
    error: str | Unset = UNSET
    project_attachment_templates: list[ProjectAttachmentTemplate] | Unset = UNSET
    resolvers: list[PluginResolver] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dir_ = self.dir_

        id = self.id

        manifest_path = self.manifest_path

        name = self.name

        trusted = self.trusted

        valid = self.valid

        version = self.version

        error = self.error

        project_attachment_templates: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.project_attachment_templates, Unset):
            project_attachment_templates = []
            for (
                project_attachment_templates_item_data
            ) in self.project_attachment_templates:
                project_attachment_templates_item = (
                    project_attachment_templates_item_data.to_dict()
                )
                project_attachment_templates.append(project_attachment_templates_item)

        resolvers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.resolvers, Unset):
            resolvers = []
            for resolvers_item_data in self.resolvers:
                resolvers_item = resolvers_item_data.to_dict()
                resolvers.append(resolvers_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "dir": dir_,
                "id": id,
                "manifestPath": manifest_path,
                "name": name,
                "trusted": trusted,
                "valid": valid,
                "version": version,
            }
        )
        if error is not UNSET:
            field_dict["error"] = error
        if project_attachment_templates is not UNSET:
            field_dict["projectAttachmentTemplates"] = project_attachment_templates
        if resolvers is not UNSET:
            field_dict["resolvers"] = resolvers

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.plugin_resolver import PluginResolver
        from ..models.project_attachment_template import ProjectAttachmentTemplate

        d = dict(src_dict)
        dir_ = d.pop("dir")

        id = d.pop("id")

        manifest_path = d.pop("manifestPath")

        name = d.pop("name")

        trusted = d.pop("trusted")

        valid = d.pop("valid")

        version = d.pop("version")

        error = d.pop("error", UNSET)

        _project_attachment_templates = d.pop("projectAttachmentTemplates", UNSET)
        project_attachment_templates: list[ProjectAttachmentTemplate] | Unset = UNSET
        if _project_attachment_templates is not UNSET:
            project_attachment_templates = []
            for project_attachment_templates_item_data in _project_attachment_templates:
                project_attachment_templates_item = ProjectAttachmentTemplate.from_dict(
                    project_attachment_templates_item_data
                )

                project_attachment_templates.append(project_attachment_templates_item)

        _resolvers = d.pop("resolvers", UNSET)
        resolvers: list[PluginResolver] | Unset = UNSET
        if _resolvers is not UNSET:
            resolvers = []
            for resolvers_item_data in _resolvers:
                resolvers_item = PluginResolver.from_dict(resolvers_item_data)

                resolvers.append(resolvers_item)

        plugin_status = cls(
            dir_=dir_,
            id=id,
            manifest_path=manifest_path,
            name=name,
            trusted=trusted,
            valid=valid,
            version=version,
            error=error,
            project_attachment_templates=project_attachment_templates,
            resolvers=resolvers,
        )

        plugin_status.additional_properties = d
        return plugin_status

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
