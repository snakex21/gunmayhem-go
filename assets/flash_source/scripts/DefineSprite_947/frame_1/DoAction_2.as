btn_continue.onRelease = function()
{
   _root.showunlocks = 0;
   _parent.slider.enableall();
   _parent.show_unlock.swapDepths(1);
   removeMovieClip(_parent.show_unlock);
   _root.playsound("menu.wav");
};
_parent.slider.disableall();
