from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.plugin_permissions import PluginPermissions
    from ..models.plugin_resolver import PluginResolver
    from ..models.plugin_review_action import PluginReviewAction
    from ..models.plugin_ui_command import PluginUICommand
    from ..models.plugin_ui_panel import PluginUIPanel


T = TypeVar("T", bound="UIContributionPlugin")


@_attrs_define
class UIContributionPlugin:
    """
    Attributes:
        enabled (bool):
        name (str):
        plugin_id (str):
        trusted (bool):
        version (str):
        commands (list[PluginUICommand] | Unset):
        disabled_reason (str | Unset):
        panels (list[PluginUIPanel] | Unset):
        permissions (None | PluginPermissions | Unset):
        resolvers (list[PluginResolver] | Unset):
        review_actions (list[PluginReviewAction] | Unset):
    """

    enabled: bool
    name: str
    plugin_id: str
    trusted: bool
    version: str
    commands: list[PluginUICommand] | Unset = UNSET
    disabled_reason: str | Unset = UNSET
    panels: list[PluginUIPanel] | Unset = UNSET
    permissions: None | PluginPermissions | Unset = UNSET
    resolvers: list[PluginResolver] | Unset = UNSET
    review_actions: list[PluginReviewAction] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.plugin_permissions import PluginPermissions

        enabled = self.enabled

        name = self.name

        plugin_id = self.plugin_id

        trusted = self.trusted

        version = self.version

        commands: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.commands, Unset):
            commands = []
            for commands_item_data in self.commands:
                commands_item = commands_item_data.to_dict()
                commands.append(commands_item)

        disabled_reason = self.disabled_reason

        panels: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.panels, Unset):
            panels = []
            for panels_item_data in self.panels:
                panels_item = panels_item_data.to_dict()
                panels.append(panels_item)

        permissions: dict[str, Any] | None | Unset
        if isinstance(self.permissions, Unset):
            permissions = UNSET
        elif isinstance(self.permissions, PluginPermissions):
            permissions = self.permissions.to_dict()
        else:
            permissions = self.permissions

        resolvers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.resolvers, Unset):
            resolvers = []
            for resolvers_item_data in self.resolvers:
                resolvers_item = resolvers_item_data.to_dict()
                resolvers.append(resolvers_item)

        review_actions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.review_actions, Unset):
            review_actions = []
            for review_actions_item_data in self.review_actions:
                review_actions_item = review_actions_item_data.to_dict()
                review_actions.append(review_actions_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "enabled": enabled,
                "name": name,
                "pluginId": plugin_id,
                "trusted": trusted,
                "version": version,
            }
        )
        if commands is not UNSET:
            field_dict["commands"] = commands
        if disabled_reason is not UNSET:
            field_dict["disabledReason"] = disabled_reason
        if panels is not UNSET:
            field_dict["panels"] = panels
        if permissions is not UNSET:
            field_dict["permissions"] = permissions
        if resolvers is not UNSET:
            field_dict["resolvers"] = resolvers
        if review_actions is not UNSET:
            field_dict["reviewActions"] = review_actions

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.plugin_permissions import PluginPermissions
        from ..models.plugin_resolver import PluginResolver
        from ..models.plugin_review_action import PluginReviewAction
        from ..models.plugin_ui_command import PluginUICommand
        from ..models.plugin_ui_panel import PluginUIPanel

        d = dict(src_dict)
        enabled = d.pop("enabled")

        name = d.pop("name")

        plugin_id = d.pop("pluginId")

        trusted = d.pop("trusted")

        version = d.pop("version")

        _commands = d.pop("commands", UNSET)
        commands: list[PluginUICommand] | Unset = UNSET
        if _commands is not UNSET:
            commands = []
            for commands_item_data in _commands:
                commands_item = PluginUICommand.from_dict(commands_item_data)

                commands.append(commands_item)

        disabled_reason = d.pop("disabledReason", UNSET)

        _panels = d.pop("panels", UNSET)
        panels: list[PluginUIPanel] | Unset = UNSET
        if _panels is not UNSET:
            panels = []
            for panels_item_data in _panels:
                panels_item = PluginUIPanel.from_dict(panels_item_data)

                panels.append(panels_item)

        def _parse_permissions(data: object) -> None | PluginPermissions | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                permissions_type_1 = PluginPermissions.from_dict(data)

                return permissions_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PluginPermissions | Unset, data)

        permissions = _parse_permissions(d.pop("permissions", UNSET))

        _resolvers = d.pop("resolvers", UNSET)
        resolvers: list[PluginResolver] | Unset = UNSET
        if _resolvers is not UNSET:
            resolvers = []
            for resolvers_item_data in _resolvers:
                resolvers_item = PluginResolver.from_dict(resolvers_item_data)

                resolvers.append(resolvers_item)

        _review_actions = d.pop("reviewActions", UNSET)
        review_actions: list[PluginReviewAction] | Unset = UNSET
        if _review_actions is not UNSET:
            review_actions = []
            for review_actions_item_data in _review_actions:
                review_actions_item = PluginReviewAction.from_dict(
                    review_actions_item_data
                )

                review_actions.append(review_actions_item)

        ui_contribution_plugin = cls(
            enabled=enabled,
            name=name,
            plugin_id=plugin_id,
            trusted=trusted,
            version=version,
            commands=commands,
            disabled_reason=disabled_reason,
            panels=panels,
            permissions=permissions,
            resolvers=resolvers,
            review_actions=review_actions,
        )

        ui_contribution_plugin.additional_properties = d
        return ui_contribution_plugin

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
