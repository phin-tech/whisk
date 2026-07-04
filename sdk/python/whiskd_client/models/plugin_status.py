from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.plugin_resolver import PluginResolver
    from ..models.plugin_usage_resolver import PluginUsageResolver
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
        registry (str | Unset):
        resolvers (list[PluginResolver] | Unset):
        usage_resolvers (list[PluginUsageResolver] | Unset):
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
    registry: str | Unset = UNSET
    resolvers: list[PluginResolver] | Unset = UNSET
    usage_resolvers: list[PluginUsageResolver] | Unset = UNSET
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

        registry = self.registry

        resolvers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.resolvers, Unset):
            resolvers = []
            for resolvers_item_data in self.resolvers:
                resolvers_item = resolvers_item_data.to_dict()
                resolvers.append(resolvers_item)

        usage_resolvers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.usage_resolvers, Unset):
            usage_resolvers = []
            for usage_resolvers_item_data in self.usage_resolvers:
                usage_resolvers_item = usage_resolvers_item_data.to_dict()
                usage_resolvers.append(usage_resolvers_item)

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
        if registry is not UNSET:
            field_dict["registry"] = registry
        if resolvers is not UNSET:
            field_dict["resolvers"] = resolvers
        if usage_resolvers is not UNSET:
            field_dict["usageResolvers"] = usage_resolvers

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.plugin_resolver import PluginResolver
        from ..models.plugin_usage_resolver import PluginUsageResolver
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

        registry = d.pop("registry", UNSET)

        _resolvers = d.pop("resolvers", UNSET)
        resolvers: list[PluginResolver] | Unset = UNSET
        if _resolvers is not UNSET:
            resolvers = []
            for resolvers_item_data in _resolvers:
                resolvers_item = PluginResolver.from_dict(resolvers_item_data)

                resolvers.append(resolvers_item)

        _usage_resolvers = d.pop("usageResolvers", UNSET)
        usage_resolvers: list[PluginUsageResolver] | Unset = UNSET
        if _usage_resolvers is not UNSET:
            usage_resolvers = []
            for usage_resolvers_item_data in _usage_resolvers:
                usage_resolvers_item = PluginUsageResolver.from_dict(
                    usage_resolvers_item_data
                )

                usage_resolvers.append(usage_resolvers_item)

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
            registry=registry,
            resolvers=resolvers,
            usage_resolvers=usage_resolvers,
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
