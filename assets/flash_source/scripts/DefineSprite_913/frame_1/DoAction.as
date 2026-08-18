gunnumber = Math.round(_rotation);
dropgun.gotoAndStop(gunnumber);
_rotation = -20;
if(_root.showunlocks != 0)
{
   this.useHandCursor = false;
}
stop();
if(_root.savedata3.data.gunarray[gunnumber - 10] == false)
{
   gotoAndStop(2);
   this.useHandCursor = false;
}
this.onRelease = function()
{
   if(_currentframe == 1)
   {
      _parent.guncard.statdisplay.gotoAndStop(gunnumber);
      _parent.guncard.dropgun.gotoAndStop(gunnumber);
   }
};
