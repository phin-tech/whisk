from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.plugin_ui_command_ref import PluginUICommandRef
    from ..models.plugin_ui_panel_entry import PluginUIPanelEntry


T = TypeVar("T", bound="PluginUIPanel")


@_attrs_define
class PluginUIPanel:
    """
    Attributes:
        id (str):
        kind (str):
        scope (str):
        title (str):
        actions (list[PluginUICommandRef] | Unset):
        entry (None | PluginUIPanelEntry | Unset):
        read (None | PluginUICommandRef | Unset):
    """

    id: str
    kind: str
    scope: str
    title: str
    actions: list[PluginUICommandRef] | Unset = UNSET
    entry: None | PluginUIPanelEntry | Unset = UNSET
    read: None | PluginUICommandRef | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.plugin_ui_command_ref import PluginUICommandRef
        from ..models.plugin_ui_panel_entry import PluginUIPanelEntry

        id = self.id

        kind = self.kind

        scope = self.scope

        title = self.title

        actions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.actions, Unset):
            actions = []
            for actions_item_data in self.actions:
                actions_item = actions_item_data.to_dict()
                actions.append(actions_item)

        entry: dict[str, Any] | None | Unset
        if isinstance(self.entry, Unset):
            entry = UNSET
        elif isinstance(self.entry, PluginUIPanelEntry):
            entry = self.entry.to_dict()
        else:
            entry = self.entry

        read: dict[str, Any] | None | Unset
        if isinstance(self.read, Unset):
            read = UNSET
        elif isinstance(self.read, PluginUICommandRef):
            read = self.read.to_dict()
        else:
            read = self.read

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "kind": kind,
                "scope": scope,
                "title": title,
            }
        )
        if actions is not UNSET:
            field_dict["actions"] = actions
        if entry is not UNSET:
            field_dict["entry"] = entry
        if read is not UNSET:
            field_dict["read"] = read

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.plugin_ui_command_ref import PluginUICommandRef
        from ..models.plugin_ui_panel_entry import PluginUIPanelEntry

        d = dict(src_dict)
        id = d.pop("id")

        kind = d.pop("kind")

        scope = d.pop("scope")

        title = d.pop("title")

        _actions = d.pop("actions", UNSET)
        actions: list[PluginUICommandRef] | Unset = UNSET
        if _actions is not UNSET:
            actions = []
            for actions_item_data in _actions:
                actions_item = PluginUICommandRef.from_dict(actions_item_data)

                actions.append(actions_item)

        def _parse_entry(data: object) -> None | PluginUIPanelEntry | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                entry_type_1 = PluginUIPanelEntry.from_dict(data)

                return entry_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PluginUIPanelEntry | Unset, data)

        entry = _parse_entry(d.pop("entry", UNSET))

        def _parse_read(data: object) -> None | PluginUICommandRef | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                read_type_1 = PluginUICommandRef.from_dict(data)

                return read_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PluginUICommandRef | Unset, data)

        read = _parse_read(d.pop("read", UNSET))

        plugin_ui_panel = cls(
            id=id,
            kind=kind,
            scope=scope,
            title=title,
            actions=actions,
            entry=entry,
            read=read,
        )

        plugin_ui_panel.additional_properties = d
        return plugin_ui_panel

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
