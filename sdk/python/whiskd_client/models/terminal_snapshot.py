from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.terminal_cursor import TerminalCursor
    from ..models.terminal_modes import TerminalModes


T = TypeVar("T", bound="TerminalSnapshot")


@_attrs_define
class TerminalSnapshot:
    """
    Attributes:
        cols (int):
        cursor (TerminalCursor):
        modes (TerminalModes):
        offset (int):
        rehydrate_sequences (str):
        rows (int):
        scrollback_ansi (str):
        viewport_ansi (str):
        mouse_encoding_modes (list[str] | Unset):
        mouse_tracking_modes (list[str] | Unset):
        rehydrate_before_viewport (str | Unset):
        title (str | Unset):
        truncated (bool | Unset):
        working_directory (str | Unset):
    """

    cols: int
    cursor: TerminalCursor
    modes: TerminalModes
    offset: int
    rehydrate_sequences: str
    rows: int
    scrollback_ansi: str
    viewport_ansi: str
    mouse_encoding_modes: list[str] | Unset = UNSET
    mouse_tracking_modes: list[str] | Unset = UNSET
    rehydrate_before_viewport: str | Unset = UNSET
    title: str | Unset = UNSET
    truncated: bool | Unset = UNSET
    working_directory: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        cols = self.cols

        cursor = self.cursor.to_dict()

        modes = self.modes.to_dict()

        offset = self.offset

        rehydrate_sequences = self.rehydrate_sequences

        rows = self.rows

        scrollback_ansi = self.scrollback_ansi

        viewport_ansi = self.viewport_ansi

        mouse_encoding_modes: list[str] | Unset = UNSET
        if not isinstance(self.mouse_encoding_modes, Unset):
            mouse_encoding_modes = self.mouse_encoding_modes

        mouse_tracking_modes: list[str] | Unset = UNSET
        if not isinstance(self.mouse_tracking_modes, Unset):
            mouse_tracking_modes = self.mouse_tracking_modes

        rehydrate_before_viewport = self.rehydrate_before_viewport

        title = self.title

        truncated = self.truncated

        working_directory = self.working_directory

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "cursor": cursor,
                "modes": modes,
                "offset": offset,
                "rehydrateSequences": rehydrate_sequences,
                "rows": rows,
                "scrollbackAnsi": scrollback_ansi,
                "viewportAnsi": viewport_ansi,
            }
        )
        if mouse_encoding_modes is not UNSET:
            field_dict["mouseEncodingModes"] = mouse_encoding_modes
        if mouse_tracking_modes is not UNSET:
            field_dict["mouseTrackingModes"] = mouse_tracking_modes
        if rehydrate_before_viewport is not UNSET:
            field_dict["rehydrateBeforeViewport"] = rehydrate_before_viewport
        if title is not UNSET:
            field_dict["title"] = title
        if truncated is not UNSET:
            field_dict["truncated"] = truncated
        if working_directory is not UNSET:
            field_dict["workingDirectory"] = working_directory

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.terminal_cursor import TerminalCursor
        from ..models.terminal_modes import TerminalModes

        d = dict(src_dict)
        cols = d.pop("cols")

        cursor = TerminalCursor.from_dict(d.pop("cursor"))

        modes = TerminalModes.from_dict(d.pop("modes"))

        offset = d.pop("offset")

        rehydrate_sequences = d.pop("rehydrateSequences")

        rows = d.pop("rows")

        scrollback_ansi = d.pop("scrollbackAnsi")

        viewport_ansi = d.pop("viewportAnsi")

        mouse_encoding_modes = cast(list[str], d.pop("mouseEncodingModes", UNSET))

        mouse_tracking_modes = cast(list[str], d.pop("mouseTrackingModes", UNSET))

        rehydrate_before_viewport = d.pop("rehydrateBeforeViewport", UNSET)

        title = d.pop("title", UNSET)

        truncated = d.pop("truncated", UNSET)

        working_directory = d.pop("workingDirectory", UNSET)

        terminal_snapshot = cls(
            cols=cols,
            cursor=cursor,
            modes=modes,
            offset=offset,
            rehydrate_sequences=rehydrate_sequences,
            rows=rows,
            scrollback_ansi=scrollback_ansi,
            viewport_ansi=viewport_ansi,
            mouse_encoding_modes=mouse_encoding_modes,
            mouse_tracking_modes=mouse_tracking_modes,
            rehydrate_before_viewport=rehydrate_before_viewport,
            title=title,
            truncated=truncated,
            working_directory=working_directory,
        )

        terminal_snapshot.additional_properties = d
        return terminal_snapshot

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
